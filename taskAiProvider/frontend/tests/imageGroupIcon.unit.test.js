import test from "node:test";
import assert from "node:assert/strict";
import {
  buildDirectUploadHeaders,
  imageGroupIconSrc,
  isSameOriginUploadURL,
  uploadImageGroupIcon,
  validateImageGroupFields,
  validateImageGroupIconFile,
} from "../src/utils/imageGroupIcon.js";

test("validateImageGroupFields 无图标时报镜像组图标必填", () => {
  assert.equal(
    validateImageGroupFields({
      name: "g",
      description: "d",
      iconFileKey: "",
      iconFile: null,
    }),
    "镜像组图标必填",
  );
});

test("validateImageGroupFields 已有 icon_file_key 且无新文件则通过", () => {
  assert.equal(
    validateImageGroupFields({
      name: "g",
      description: "d",
      iconFileKey: "1/image_group_icon_2.png",
      iconFile: null,
    }),
    "",
  );
});

test("validateImageGroupIconFile 拒绝超大文件", () => {
  const file = { name: "a.png", type: "image/png", size: 512 * 1024 + 1 };
  assert.equal(
    validateImageGroupIconFile(file),
    "图标须为 PNG/JPEG/WEBP 且不超过 512KB",
  );
});

test("imageGroupIconSrc 读取组或嵌套 image_group.icon_url", () => {
  assert.equal(
    imageGroupIconSrc({ icon_url: "/api/public/image-groups/1/icon?h=ab" }),
    "/api/public/image-groups/1/icon?h=ab",
  );
  assert.equal(imageGroupIconSrc({ image_group: { icon_url: "/x" } }), "/x");
  assert.equal(imageGroupIconSrc({}), "");
});

test("isSameOriginUploadURL 仅相对路径为同源", () => {
  assert.equal(
    isSameOriginUploadURL(
      "/api/vendor/image-groups/icon-local-put/?file_key=1",
    ),
    true,
  );
  assert.equal(
    isSameOriginUploadURL(
      "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/x.png",
    ),
    false,
  );
  assert.equal(isSameOriginUploadURL("//evil.example/steal"), false);
  assert.equal(isSameOriginUploadURL(""), false);
});

test("buildDirectUploadHeaders COS 预签名不带 Authorization", () => {
  const headers = buildDirectUploadHeaders({
    issuedHeaders: {
      "Content-Type": "image/png",
      "x-cos-server-side-encryption": "AES256",
    },
    uploadURL:
      "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/icon.png?q-sign-algorithm=sha1",
    bearerToken: "vendor-secret-token",
  });
  assert.equal(headers.Authorization, undefined);
  assert.equal(headers["Content-Type"], "image/png");
  assert.equal(headers["x-cos-server-side-encryption"], "AES256");
});

test("buildDirectUploadHeaders 同源 local-put 才带 Bearer", () => {
  const headers = buildDirectUploadHeaders({
    issuedHeaders: { "Content-Type": "image/png" },
    uploadURL: "/api/vendor/image-groups/icon-local-put/?file_key=1",
    bearerToken: "vendor-secret-token",
  });
  assert.equal(headers.Authorization, "Bearer vendor-secret-token");
});

test("uploadImageGroupIcon COS POST FormData 且 complete", async () => {
  const origLS = globalThis.localStorage;
  globalThis.localStorage = { getItem: () => "vendor-secret-token" };
  const posts = [];
  const complete = [];
  const file = { name: "a.png", type: "image/png", size: 8 };
  const api = async (path) => {
    if (String(path).includes("icon-upload-url")) {
      return {
        file_key: "k1",
        upload_url:
          "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/",
        method: "POST",
        form_fields: {
          key: "k1",
          policy: "cG9saWN5",
          "q-signature": "ab",
          "x-cos-server-side-encryption": "AES256",
        },
        traceId: "tid-upload-url",
      };
    }
    complete.push(path);
    return { file_key: "k1" };
  };
  const fetchImpl = async (url, opts) => {
    posts.push({ url, opts });
    return { ok: false, type: "opaque", status: 0, headers: { get: () => "" } };
  };
  try {
    const key = await uploadImageGroupIcon(file, { api, fetchImpl });
    assert.equal(key, "k1");
    assert.equal(posts.length, 1);
    assert.equal(posts[0].opts.mode, "no-cors");
    assert.equal(posts[0].opts.body instanceof FormData, true);
    assert.equal(posts[0].opts.headers, undefined);
    assert.equal(complete.length, 1);
  } finally {
    if (origLS === undefined) delete globalThis.localStorage;
    else globalThis.localStorage = origLS;
  }
});

test("uploadImageGroupIcon 同源 PUT Failed to fetch 挂上 upload-url 的 traceId", async () => {
  const file = { name: "a.png", type: "image/png", size: 8 };
  const api = async () => ({
    file_key: "k1",
    upload_url: "/api/vendor/image-groups/icon-local-put/?file_key=k1",
    method: "PUT",
    headers: { "Content-Type": "image/png" },
    traceId: "tid-upload-url",
  });
  const fetchImpl = async () => {
    throw new TypeError("Failed to fetch");
  };
  await assert.rejects(
    () => uploadImageGroupIcon(file, { api, fetchImpl }),
    (err) => {
      assert.equal(err.message, "Failed to fetch");
      assert.equal(err.traceId, "tid-upload-url");
      return true;
    },
  );
});
