import test from "node:test";
import assert from "node:assert/strict";
import {
  buildDirectUploadHeaders,
  hasPostFormFields,
  isSameOriginUploadURL,
  uploadIssuedFile,
} from "../src/utils/directUpload.js";

test("hasPostFormFields 仅在 form_fields 非空时为真", () => {
  assert.equal(hasPostFormFields({ form_fields: { key: "k" } }), true);
  assert.equal(hasPostFormFields({ form_fields: {} }), false);
  assert.equal(hasPostFormFields({}), false);
  assert.equal(hasPostFormFields(null), false);
});

test("uploadIssuedFile COS POST 用 FormData、no-cors、不带 Authorization", async () => {
  const posts = [];
  const file = { name: "a.png", type: "image/png", size: 8 };
  const issued = {
    file_key: "k1",
    upload_url: "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/",
    method: "POST",
    form_fields: {
      key: "k1",
      policy: "cG9saWN5",
      "q-sign-algorithm": "sha1",
      "q-ak": "AKIDxx",
      "q-key-time": "1;2",
      "q-signature": "deadbeef",
      "x-cos-server-side-encryption": "AES256",
    },
    headers: { "Content-Type": "image/png" },
    traceId: "tid-post",
  };
  const fetchImpl = async (url, opts) => {
    posts.push({ url, opts });
    return { ok: false, type: "opaque", status: 0, headers: { get: () => "" } };
  };
  await uploadIssuedFile(file, issued, {
    fetchImpl,
    bearerToken: "vendor-secret-token",
  });
  assert.equal(posts.length, 1);
  assert.equal(posts[0].url, issued.upload_url);
  assert.equal(posts[0].opts.method, "POST");
  assert.equal(posts[0].opts.mode, "no-cors");
  assert.equal(posts[0].opts.credentials, "omit");
  assert.equal(posts[0].opts.headers, undefined);
  assert.equal(posts[0].opts.body instanceof FormData, true);
  assert.equal(posts[0].opts.body.get("key"), "k1");
  assert.equal(posts[0].opts.body.has("file"), true);
});

test("uploadIssuedFile COS POST Failed to fetch 不阻断（complete Head 确认）", async () => {
  const issued = {
    upload_url: "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/",
    method: "POST",
    form_fields: { key: "k1", policy: "p" },
    traceId: "tid-post",
  };
  const fetchImpl = async () => {
    throw new TypeError("Failed to fetch");
  };
  await uploadIssuedFile({ name: "a.png", size: 1 }, issued, { fetchImpl });
});

test("uploadIssuedFile 同源 PUT 带 Bearer 且校验 ok", async () => {
  const puts = [];
  const issued = {
    upload_url: "/api/vendor/image-groups/icon-local-put/?file_key=1",
    method: "PUT",
    headers: { "Content-Type": "image/png" },
  };
  const fetchImpl = async (url, opts) => {
    puts.push({ url, opts });
    return { ok: true, headers: { get: () => "" } };
  };
  await uploadIssuedFile({ name: "a.png", size: 8 }, issued, {
    fetchImpl,
    bearerToken: "vendor-secret-token",
  });
  assert.equal(puts[0].opts.credentials, "include");
  assert.equal(puts[0].opts.headers.Authorization, "Bearer vendor-secret-token");
  assert.equal(puts[0].opts.body.name, "a.png");
});

test("isSameOriginUploadURL 仅相对路径为同源", () => {
  assert.equal(isSameOriginUploadURL("/api/vendor/application/local-put/?k=1"), true);
  assert.equal(
    isSameOriginUploadURL(
      "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/",
    ),
    false,
  );
});

test("buildDirectUploadHeaders COS URL 不带 Authorization", () => {
  const headers = buildDirectUploadHeaders({
    issuedHeaders: { "Content-Type": "image/png" },
    uploadURL: "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/",
    bearerToken: "vendor-secret-token",
  });
  assert.equal(headers.Authorization, undefined);
});
