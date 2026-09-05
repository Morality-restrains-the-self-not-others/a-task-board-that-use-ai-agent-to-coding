import test from "node:test";
import assert from "node:assert/strict";
import {
  buildRegionEnvServerOptions,
  serverImageMatchesArch,
  serverImageMatchesRegion,
} from "../src/utils/regionEnvServerOptions.js";

const region = { id: "cn-hangzhou", name: "华东1（杭州）" };

const images = [
  {
    id: "1",
    platform_type: "aliyun",
    region: "cn-beijing",
    image_name: "Beijing Ubuntu",
    architecture: "x86_64",
    is_active: true,
  },
  {
    id: "2",
    platform_type: "aliyun",
    region: "cn-hangzhou",
    image_name: "Hangzhou ARM",
    architecture: "arm64",
    is_active: true,
  },
  {
    id: "3",
    platform_type: "aliyun",
    region: "cn-hangzhou",
    image_name: "Hangzhou Ubuntu",
    architecture: "x86_64",
    is_active: true,
  },
  {
    id: "4",
    platform_type: "tencentcloud",
    region: "ap-guangzhou",
    image_name: "Other Cloud",
    architecture: "x86_64",
    is_active: true,
  },
  {
    id: "5",
    platform_type: "aliyun",
    region: "cn-hangzhou",
    image_name: "Disabled",
    architecture: "x86_64",
    is_active: false,
  },
];

test("serverImageMatchesRegion 支持 region id 与 name", () => {
  assert.equal(serverImageMatchesRegion({ region: "cn-hangzhou" }, region), true);
  assert.equal(serverImageMatchesRegion({ region: "华东1（杭州）" }, region), true);
  assert.equal(serverImageMatchesRegion({ region: "cn-beijing" }, region), false);
});

test("buildRegionEnvServerOptions 返回同云平台全部启用镜像", () => {
  const options = buildRegionEnvServerOptions(images, {
    platform: "aliyun",
    region,
    targetArchs: ["amd64"],
    selectedId: "",
  });
  assert.deepEqual(options.map((x) => x.id), ["3", "2", "1"]);
});

test("buildRegionEnvServerOptions 优先本区域与匹配架构", () => {
  const options = buildRegionEnvServerOptions(images, {
    platform: "aliyun",
    region,
    targetArchs: ["arm64"],
    selectedId: "",
  });
  assert.equal(options[0].id, "2");
  assert.equal(options[1].id, "3");
});

test("buildRegionEnvServerOptions 保留当前选中项", () => {
  const options = buildRegionEnvServerOptions(images, {
    platform: "aliyun",
    region,
    targetArchs: [],
    selectedId: "5",
  });
  assert.equal(options.at(-1)?.id, "5");
});

test("serverImageMatchesArch 空架构视为兼容", () => {
  assert.equal(serverImageMatchesArch({ architecture: "" }, ["amd64"]), true);
  assert.equal(serverImageMatchesArch({ architecture: "arm64" }, ["amd64"]), false);
});
