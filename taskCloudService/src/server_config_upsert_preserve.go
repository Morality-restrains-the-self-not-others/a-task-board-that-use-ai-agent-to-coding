package main

// cloudServerRuntimeUpsertPreserve 防止 ensure/heal 用空 struct 撞 UNIQUE
// (workspace_id,task_id,comment_id) 时抹掉已绑定的运行态。
// 评论行 + 空 instance + 非终态 status → 保留旧 instance/IP/URL/status。
// 显式 Released/Stopped/Terminated/Stopping 或任务级（comment_id 空）允许清空。
const cloudServerRuntimeUpsertPreserve = `
			instance_id=CASE
				WHEN TRIM(COALESCE(VALUES(instance_id),'')) != '' THEN VALUES(instance_id)
				WHEN LOWER(TRIM(COALESCE(VALUES(last_runtime_status),''))) IN ('released','stopped','terminated','stopping') THEN VALUES(instance_id)
				WHEN TRIM(COALESCE(VALUES(comment_id), cloud_server_configs.comment_id, '')) != '' THEN cloud_server_configs.instance_id
				ELSE VALUES(instance_id)
			END,
			public_ip=CASE
				WHEN TRIM(COALESCE(VALUES(public_ip),'')) != '' THEN VALUES(public_ip)
				WHEN LOWER(TRIM(COALESCE(VALUES(last_runtime_status),''))) IN ('released','stopped','terminated','stopping') THEN VALUES(public_ip)
				WHEN TRIM(COALESCE(VALUES(comment_id), cloud_server_configs.comment_id, '')) != '' THEN cloud_server_configs.public_ip
				ELSE VALUES(public_ip)
			END,
			server_url=CASE
				WHEN TRIM(COALESCE(VALUES(server_url),'')) != '' THEN VALUES(server_url)
				WHEN LOWER(TRIM(COALESCE(VALUES(last_runtime_status),''))) IN ('released','stopped','terminated','stopping') THEN VALUES(server_url)
				WHEN TRIM(COALESCE(VALUES(comment_id), cloud_server_configs.comment_id, '')) != '' THEN cloud_server_configs.server_url
				ELSE VALUES(server_url)
			END,
			last_runtime_status=CASE
				WHEN TRIM(COALESCE(VALUES(last_runtime_status),'')) != '' THEN VALUES(last_runtime_status)
				WHEN TRIM(COALESCE(VALUES(comment_id), cloud_server_configs.comment_id, '')) != '' THEN cloud_server_configs.last_runtime_status
				ELSE VALUES(last_runtime_status)
			END,
`
