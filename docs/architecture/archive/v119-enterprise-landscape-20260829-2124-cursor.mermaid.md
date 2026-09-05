# v119 enterprise-landscape — 多区域镜像仓库副本 (target)

```mermaid
graph TD;
  vendor["镜像厂商 NEW"];
  commenter["发评 / 自动运行"];
  declare["按区域登记公网仓库 NEW"];
  startVm["评论启动云服务器 MODIFIED"];
  portal["厂商门户"];
  aip["taskAiProvider MODIFIED"];
  cloud["taskCloudService MODIFIED"];
  regPub["区域公网 Registry"];
  regVpc["区域内网 Registry NEW"];
  p118["Plateau v118"];
  p119["Plateau v119"];
  gap["Gap: 单 image_url 跨区拉取"];
  wp["WP-multi-region-image-replicas"];
  vendor --> declare;
  commenter --> startVm;
  vendor --> portal;
  portal --> aip;
  commenter --> cloud;
  cloud --> aip;
  regPub --> aip;
  regVpc --> startVm;
  p118 --> gap;
  wp --|> gap;
  wp --|> p119;
```
