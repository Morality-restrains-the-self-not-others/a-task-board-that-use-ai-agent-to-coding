# v119 application-integration — 多区域镜像仓库副本 (target)

```mermaid
graph TD;
  portal["厂商门户 Vue"];
  gw["taskGateway / APISIX"];
  aip["taskAiProvider MODIFIED"];
  cloud["taskCloudService MODIFIED"];
  ev["taskEvents"];
  fe["taskFE SPA"];
  img["vendorcontainerimage.image_url"];
  assoc["containercloudserverassociation"];
  replica["containerimage_registry_replica NEW"];
  installed["cloud_tenant_installed_images MODIFIED"];
  evR["ContainerImageReplicasUpdated NEW"];
  evS["CLOUD_SERVER_STARTED MODIFIED"];
  p118["Plateau v118 单 URL"];
  p119["Plateau v119 区域副本"];
  gap["Gap: 启机跨区拉规范仓库"];
  wp["WP-multi-region-image-replicas"];
  portal --> gw;
  gw --> aip;
  aip --> img;
  aip --> assoc;
  aip --> replica;
  aip --> evR;
  fe --> gw;
  gw --> cloud;
  cloud --> installed;
  aip --> cloud;
  cloud --> evS;
  cloud --> ev;
  p118 --> gap;
  wp --|> gap;
  wp --|> p119;
```
