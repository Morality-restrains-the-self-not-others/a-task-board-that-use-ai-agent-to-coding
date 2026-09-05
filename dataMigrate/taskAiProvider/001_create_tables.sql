-- ai-provider (taskAiProvider) MySQL schema
-- Replaces the old Django manage.py migrate from Saas_Ai_Provider

CREATE TABLE IF NOT EXISTS ai_provider_vendor (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    saas_user_id BIGINT,
    email VARCHAR(255),
    password_hash VARCHAR(255),
    company_name VARCHAR(255),
    contact_name VARCHAR(255),
    is_active TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_provider_containerimagegroup (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    vendor_id BIGINT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_provider_vendorcontainerimage (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    version VARCHAR(255),
    image_url TEXT,
    target_architectures TEXT,
    size BIGINT,
    status VARCHAR(64),
    review_note TEXT,
    reviewed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    image_group_id BIGINT,
    reviewer_id BIGINT,
    vendor_id BIGINT,
    auto_run_steps_md TEXT,
    auto_run_steps_extract_status VARCHAR(64),
    auto_run_steps_digest VARCHAR(255)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_provider_userdatatemplate (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255),
    version VARCHAR(64),
    os_type VARCHAR(64),
    variables TEXT,
    container_variables TEXT,
    content TEXT,
    auto_verify_script TEXT,
    is_active TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_provider_vendorcloudserverimage (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    vendor_id BIGINT,
    platform_type VARCHAR(64),
    image_name VARCHAR(255),
    image_id VARCHAR(255),
    region VARCHAR(64),
    os_type VARCHAR(64),
    os_version VARCHAR(64),
    architecture VARCHAR(64),
    image_type VARCHAR(64),
    image_size_gb INT,
    is_active TINYINT DEFAULT 0,
    default_instance_type_id VARCHAR(255),
    default_instance_type_label VARCHAR(255),
    base_cpu_cores INT,
    base_memory_gib INT,
    userdata_template_id BIGINT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_provider_containercloudserverassociation (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    platform_type VARCHAR(64),
    region VARCHAR(64),
    cloud_server_image_id BIGINT,
    container_image_id BIGINT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_provider_vendorcloudserverimageuserdata (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    cloud_server_image_id BIGINT,
    userdata_run_verified DATETIME,
    verification_secret VARCHAR(255)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Platform staff table (SSO staff bridge — superadmin auto-provisioned on first access)
-- Used by taskAiProvider store_auth.go for staff authentication
CREATE TABLE IF NOT EXISTS ai_provider_platformstaff (
    id BIGINT PRIMARY KEY,
    username VARCHAR(255),
    password_hash VARCHAR(255),
    display_name VARCHAR(255),
    is_active TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    saas_superadmin_id BIGINT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
