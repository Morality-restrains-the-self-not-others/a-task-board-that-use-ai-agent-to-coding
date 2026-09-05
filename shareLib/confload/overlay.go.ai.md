# overlay.go Companion

`MergeConfLocal` / `ReadYAMLMerged` / `UnmarshalYAMLMerged` / `MergeYAMLAtPath` 叠 conf-local（ADR-0054）。只读本 app 相对路径，不引入跨服务 conf 直读。`docker-infra.yaml` 片段须经 `mergeOptionalFragment` 叠 conf-local，禁止只读 tracked。
