import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  autoRunStatusHint,
  hasPersistedResolveInfo,
  parseResolvePayload,
  persistedResolveInfo,
  resolveRefKey,
  skillsStatusHint,
  summarizeImageResolveInfo,
} from "../src/lib/imageResolveInfo.js";

describe("resolveRefKey", () => {
  it("combines trimmed image url and version", () => {
    assert.equal(resolveRefKey("  nginx:1.25  ", "1.25"), "nginx:1.25@1.25");
  });
  it("keeps empty version", () => {
    assert.equal(resolveRefKey("nginx", ""), "nginx@");
  });
  it("treats nulls as empty strings", () => {
    assert.equal(resolveRefKey(null, undefined), "@");
  });
});

describe("parseResolvePayload", () => {
  it("extracts skills and auto run fields from full payload", () => {
    const info = parseResolvePayload({
      target_architectures: ["x86_64"],
      size: 123,
      skills: {
        version: 1,
        default_skill: "coding",
        skills: [
          { name: "coding", description: "通用编码", is_default: true },
          { name: "review", description: "代码审查", is_default: false },
        ],
      },
      skills_status: "ok",
      skills_detail: "",
      auto_run_steps_md: "# 自动运行\n1. 启动服务",
      auto_run_steps_status: "ok",
      auto_run_steps_detail: "",
    });
    assert.equal(info.skills.length, 2);
    assert.equal(info.skills[0].name, "coding");
    assert.equal(info.skills[0].is_default, true);
    assert.equal(info.skillsStatus, "ok");
    assert.equal(info.skillsDetail, "");
    assert.match(info.autoRunStepsMd, /启动服务/);
    assert.equal(info.autoRunStepsStatus, "ok");
  });

  it("handles missing / null payload fields", () => {
    const info = parseResolvePayload({ target_architectures: ["arm64"] });
    assert.deepEqual(info.skills, []);
    assert.equal(info.skillsStatus, "");
    assert.equal(info.autoRunStepsMd, "");
    assert.equal(info.autoRunStepsStatus, "");
  });

  it("tolerates null payload", () => {
    const info = parseResolvePayload(null);
    assert.deepEqual(info.skills, []);
    assert.equal(info.skillsStatus, "");
  });

  it("tolerates skills being a non-object", () => {
    const info = parseResolvePayload({ skills: "unexpected" });
    assert.deepEqual(info.skills, []);
  });

  it("keeps non-ok statuses and details", () => {
    const info = parseResolvePayload({
      skills_status: "auth_failed",
      skills_detail: "auth_failed: unauthorized",
      auto_run_steps_status: "not_found",
      auto_run_steps_detail: "autoRunStep.md not found in image layers",
    });
    assert.equal(info.skillsStatus, "auth_failed");
    assert.equal(info.skillsDetail, "auth_failed: unauthorized");
    assert.equal(info.autoRunStepsStatus, "not_found");
  });
});

describe("hasPersistedResolveInfo", () => {
  it("true only when both extract statuses are ok", () => {
    assert.equal(
      hasPersistedResolveInfo({
        image_skills_extract_status: "ok",
        auto_run_steps_extract_status: "ok",
      }),
      true,
    );
    assert.equal(
      hasPersistedResolveInfo({
        image_skills_extract_status: "ok",
        auto_run_steps_extract_status: "pending",
      }),
      false,
    );
    assert.equal(
      hasPersistedResolveInfo({
        image_skills_extract_status: "not_found",
        auto_run_steps_extract_status: "ok",
      }),
      false,
    );
  });
  it("false for null / empty record", () => {
    assert.equal(hasPersistedResolveInfo(null), false);
    assert.equal(hasPersistedResolveInfo(undefined), false);
    assert.equal(hasPersistedResolveInfo({}), false);
  });
});

describe("persistedResolveInfo", () => {
  it("builds resolve info from persisted skills and auto run when both ok", () => {
    const info = persistedResolveInfo({
      image_skills_extract_status: "ok",
      image_skills: [
        { name: "coding", description: "通用编码", is_default: true },
        { name: "review", description: "代码审查", is_default: false },
      ],
      auto_run_steps_extract_status: "ok",
      auto_run_steps_md: "# 自动运行\n1. 启动服务",
    });
    assert.ok(info, "expected non-null");
    assert.equal(info.skills.length, 2);
    assert.equal(info.skills[0].name, "coding");
    assert.equal(info.skillsStatus, "ok");
    assert.match(info.autoRunStepsMd, /启动服务/);
    assert.equal(info.autoRunStepsStatus, "ok");
  });
  it("returns null when either field not fully extracted", () => {
    assert.equal(
      persistedResolveInfo({ image_skills_extract_status: "ok", auto_run_steps_extract_status: "pending" }),
      null,
    );
    assert.equal(
      persistedResolveInfo({ image_skills_extract_status: "failed", auto_run_steps_extract_status: "ok" }),
      null,
    );
  });
  it("tolerates missing image_skills / auto_run_steps_md when statuses ok", () => {
    const info = persistedResolveInfo({
      image_skills_extract_status: "ok",
      auto_run_steps_extract_status: "ok",
    });
    assert.deepEqual(info.skills, []);
    assert.equal(info.autoRunStepsMd, "");
  });
});

describe("skillsStatusHint", () => {
  it("maps known statuses to hints", () => {
    assert.equal(skillsStatusHint("not_found"), "镜像中未发现 imageSkills.yaml");
    assert.equal(skillsStatusHint("auth_failed"), "镜像仓库认证失败，未能提取技能列表");
    assert.equal(skillsStatusHint("failed"), "技能列表提取失败");
  });
  it("returns empty for ok and unknown", () => {
    assert.equal(skillsStatusHint("ok"), "");
    assert.equal(skillsStatusHint(""), "");
    assert.equal(skillsStatusHint("weird"), "");
  });
});

describe("autoRunStatusHint", () => {
  it("maps known statuses to hints", () => {
    assert.equal(autoRunStatusHint("not_found"), "镜像中未发现 autoRunStep.md");
    assert.equal(autoRunStatusHint("auth_failed"), "镜像仓库认证失败，未能提取自动运行说明");
    assert.equal(autoRunStatusHint("failed"), "自动运行说明提取失败");
  });
  it("returns empty for ok and unknown", () => {
    assert.equal(autoRunStatusHint("ok"), "");
    assert.equal(autoRunStatusHint(""), "");
  });
});

describe("summarizeImageResolveInfo (OPT-20260824-063)", () => {
  it("summarizes ok skills and auto run first line", () => {
    const s = summarizeImageResolveInfo({
      image_skills_extract_status: "ok",
      image_skills: [
        { name: "coding", description: "通用编码", is_default: true },
        { name: "review", description: "代码审查", is_default: false },
      ],
      auto_run_steps_extract_status: "ok",
      auto_run_steps_md: "# 自动运行\n1. 启动服务\n2. 汇报结果",
    });
    assert.deepEqual(s.skillNames, ["coding", "review"]);
    assert.equal(s.skillsHint, "");
    assert.equal(s.autoRunPreview, "启动服务");
    assert.equal(s.autoRunHint, "");
  });

  it("ok but empty skills/auto-run → 镜像未声明 hints", () => {
    const s = summarizeImageResolveInfo({
      image_skills_extract_status: "ok",
      image_skills: [],
      auto_run_steps_extract_status: "ok",
      auto_run_steps_md: "",
    });
    assert.deepEqual(s.skillNames, []);
    assert.equal(s.skillsHint, "镜像未声明技能");
    assert.equal(s.autoRunPreview, "");
    assert.equal(s.autoRunHint, "镜像未声明自动运行说明");
  });

  it("pending/empty extract status → 未提取 hints", () => {
    const s = summarizeImageResolveInfo({});
    assert.deepEqual(s.skillNames, []);
    assert.equal(s.skillsHint, "技能未提取");
    assert.equal(s.autoRunPreview, "");
    assert.equal(s.autoRunHint, "运行说明未提取");
  });

  it("known failure statuses surface failure hints", () => {
    const s = summarizeImageResolveInfo({
      image_skills_extract_status: "auth_failed",
      auto_run_steps_extract_status: "not_found",
    });
    assert.equal(s.skillsHint, "镜像仓库认证失败，未能提取技能列表");
    assert.equal(s.autoRunHint, "镜像中未发现 autoRunStep.md");
  });

  it("tolerates null record and non-array image_skills", () => {
    const s = summarizeImageResolveInfo(null);
    assert.deepEqual(s.skillNames, []);
    assert.equal(s.skillsHint, "技能未提取");

    const s2 = summarizeImageResolveInfo({
      image_skills_extract_status: "ok",
      image_skills: "unexpected",
    });
    assert.deepEqual(s2.skillNames, []);
    assert.equal(s2.skillsHint, "镜像未声明技能");
  });
});
