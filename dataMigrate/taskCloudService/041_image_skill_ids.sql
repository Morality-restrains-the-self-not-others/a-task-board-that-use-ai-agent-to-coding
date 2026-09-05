-- OPT-20260824-078 已存镜像 image_skills_json 技能补派生 ID（D1=B 服务端派生）。
--
-- 背景：创建任务需要以技能 ID 落库（task_tasks.image_skill_id），但存量行的
--       image_skills_json 技能只有 {name, description, is_default}，无 id 字段。
--       新提取/安装路径已写入 id（sk_<sha1(name+seed)>[:12]）；本迁移按与 Go 侧
--       deriveImageSkillID 完全一致的规则回填：seed = COALESCE(NULLIF(external_id,''), id)。
--
-- 幂等：JSON_SET 前 IFNULL 保留已带非空 id 的技能；重复执行结果不变。
UPDATE `cloud_tenant_installed_images`
SET `image_skills_json` = JSON_OBJECT(
  'version', JSON_EXTRACT(`image_skills_json`, '$.version'),
  'default_skill', JSON_EXTRACT(`image_skills_json`, '$.default_skill'),
  'skills', (
    SELECT JSON_ARRAYAGG(
      JSON_SET(skill, '$.id',
        IFNULL(
          NULLIF(JSON_UNQUOTE(JSON_EXTRACT(skill, '$.id')), ''),
          CONCAT('sk_', LEFT(SHA1(CONCAT(
            JSON_UNQUOTE(JSON_EXTRACT(skill, '$.name')),
            COALESCE(NULLIF(`external_image_id`, ''), `id`)
          )), 12))
        )
      )
    )
    FROM JSON_TABLE(`image_skills_json`, '$.skills[*]' COLUMNS (skill JSON PATH '$')) AS jt
  )
)
WHERE `image_skills_json` IS NOT NULL
  AND JSON_LENGTH(`image_skills_json`, '$.skills') > 0;
