-- taskCloudService: Rebuild cloud_tenant_installed_images for very old databases
-- Extracted from Go migrateTenantInstalledImagesTable()
-- Guard: external_image_id column is missing OR legacy company_id column still exists.
-- On modern databases (001_schema.sql →) both conditions are false → no-op.
-- Applied once via data_migrate_log tracking.

-- Check rebuild condition BEFORE any DDL (persisted in @needs_rebuild for subsequent statements)
SET @needs_rebuild = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_tenant_installed_images' AND COLUMN_NAME = 'external_image_id') = 0
  OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_tenant_installed_images' AND COLUMN_NAME = 'company_id') > 0,
  1, 0
);

-- Drop old table (destructive — data loss is acceptable: this only fires on pre-migration databases)
SET @stmt = IF(@needs_rebuild = 1, 'DROP TABLE IF EXISTS cloud_tenant_installed_images', 'SELECT 1 \'skip: cloud_tenant_installed_images rebuild not needed\'');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Re-create with correct modern schema (matching 001_schema.sql)
SET @stmt = IF(@needs_rebuild = 1, 'CREATE TABLE IF NOT EXISTS cloud_tenant_installed_images (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    external_image_id VARCHAR(255) NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    version VARCHAR(64) DEFAULT \'\',
    image_url TEXT NOT NULL,
    target_architectures TEXT,
    size INTEGER,
    is_dev_mode TINYINT DEFAULT 0,
    vendor_id VARCHAR(64) DEFAULT \'\',
    vendor_name VARCHAR(255) DEFAULT \'\',
    installed_by_id VARCHAR(64) DEFAULT \'\',
    installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    userdata_template_id VARCHAR(64) DEFAULT \'\',
    auto_run_steps_md TEXT,
    auto_run_steps_extract_status VARCHAR(64) DEFAULT \'\',
    auto_run_steps_digest VARCHAR(255) DEFAULT \'\',
    UNIQUE(tenant_id, external_image_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Create index (matching 001_schema.sql)
SET @stmt = IF(@needs_rebuild = 1, 'CREATE INDEX idx_tii_tenant ON cloud_tenant_installed_images(tenant_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
