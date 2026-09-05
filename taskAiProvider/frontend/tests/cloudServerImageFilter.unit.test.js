import test from "node:test";
import assert from "node:assert/strict";
import { filterCloudServerImagesByKeyword } from "../src/utils/cloudServerImageFilter.js";

test("空关键词返回全部镜像", () => {
  const images = [
    { id: "m-1", name: "Ubuntu 22.04", architecture: "x86_64" },
    { id: "m-2", name: "CentOS 7", architecture: "arm64" },
  ];
  const result = filterCloudServerImagesByKeyword(images, "   ");
  assert.equal(result.length, 2);
  assert.deepEqual(result.map((x) => x.id), ["m-1", "m-2"]);
});

test("可按镜像名称和镜像 ID 过滤", () => {
  const images = [
    { id: "m-ubuntu-001", name: "Ubuntu 22.04 LTS", os_type: "linux", architecture: "x86_64" },
    { id: "m-centos-001", name: "CentOS 7", os_type: "linux", architecture: "x86_64" },
  ];
  const byName = filterCloudServerImagesByKeyword(images, "ubuntu");
  const byId = filterCloudServerImagesByKeyword(images, "centos-001");

  assert.deepEqual(byName.map((x) => x.id), ["m-ubuntu-001"]);
  assert.deepEqual(byId.map((x) => x.id), ["m-centos-001"]);
});

test("可按系统字段与架构过滤，且大小写不敏感", () => {
  const images = [
    { id: "m-win", name: "Windows", os_type: "windows", os_version: "server_2022", architecture: "x86_64" },
    { id: "m-arm", name: "Debian", os_type: "linux", os_version: "debian_12", architecture: "arm64" },
  ];
  const byOsVersion = filterCloudServerImagesByKeyword(images, "SERVER_2022");
  const byArch = filterCloudServerImagesByKeyword(images, "ARM64");

  assert.deepEqual(byOsVersion.map((x) => x.id), ["m-win"]);
  assert.deepEqual(byArch.map((x) => x.id), ["m-arm"]);
});
