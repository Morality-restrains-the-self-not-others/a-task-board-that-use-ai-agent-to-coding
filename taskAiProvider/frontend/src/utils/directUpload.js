/**
 * Browser upload for issued presign: COS POST Object (FormData) or same-origin PUT.
 * OPT-20260901-008：与 taskFE/app/src/utils/vendorDocsDirectUpload.js 为契约孪生，
 * 两仓独立无法共享文件——修改直传契约时须同步两文件；两侧单测同构断言同契约点。
 */

import { attachClientTraceId } from "./traceId.js";

/** COS 预签名是跨域绝对 URL；仅本服务 local-put 相对路径才算同源。 */
export function isSameOriginUploadURL(uploadURL) {
  return (
    typeof uploadURL === "string" &&
    uploadURL.startsWith("/") &&
    !uploadURL.startsWith("//")
  );
}

export function hasPostFormFields(issued) {
  const fields = issued && issued.form_fields;
  return (
    !!fields &&
    typeof fields === "object" &&
    !Array.isArray(fields) &&
    Object.keys(fields).length > 0
  );
}

/**
 * 浏览器直传 COS 禁止带 Authorization：会触发 OPTIONS 预检要 authorization，
 * 桶 CORS 不含该头则 403，且 query-string 签名也不包含 Bearer。
 */
export function buildDirectUploadHeaders({
  issuedHeaders,
  uploadURL,
  bearerToken,
} = {}) {
  const headers = { ...(issuedHeaders || {}) };
  if (isSameOriginUploadURL(uploadURL) && bearerToken) {
    headers.Authorization = `Bearer ${bearerToken}`;
  }
  return headers;
}

function isOpaqueCorsFetchError(err) {
  return (
    err instanceof TypeError && /failed to fetch/i.test(String(err.message || ""))
  );
}

/**
 * COS POST Object：multipart + 无自定义头，避免 OPTIONS。
 * 桶无 CORS 时响应不可读（opaque / Failed to fetch），对象是否写入由后续 complete Head 确认。
 */
export async function uploadIssuedFile(file, issued, { fetchImpl, bearerToken } = {}) {
  const doFetch = fetchImpl || fetch;
  const uploadURL = issued && issued.upload_url;
  if (!uploadURL) {
    throw new Error("缺少上传地址");
  }
  if (hasPostFormFields(issued)) {
    const fd = new FormData();
    for (const [k, v] of Object.entries(issued.form_fields)) {
      fd.append(k, String(v));
    }
    fd.append("file", file);
    try {
      await doFetch(uploadURL, {
        method: issued.method || "POST",
        body: fd,
        credentials: "omit",
        mode: "no-cors",
      });
    } catch (networkErr) {
      const err =
        networkErr instanceof Error ? networkErr : new Error(String(networkErr));
      attachClientTraceId(err, issued.traceId);
      if (isOpaqueCorsFetchError(err)) {
        return;
      }
      throw err;
    }
    return;
  }
  const headers = buildDirectUploadHeaders({
    issuedHeaders: (issued && issued.headers) || {},
    uploadURL,
    bearerToken,
  });
  const sameOrigin = isSameOriginUploadURL(uploadURL);
  let putRes;
  try {
    putRes = await doFetch(uploadURL, {
      method: (issued && issued.method) || "PUT",
      headers,
      body: file,
      credentials: sameOrigin ? "include" : "omit",
    });
  } catch (networkErr) {
    const err =
      networkErr instanceof Error ? networkErr : new Error(String(networkErr));
    attachClientTraceId(err, issued && issued.traceId);
    throw err;
  }
  if (!putRes.ok) {
    const e = new Error("直传失败");
    e.traceId = putRes.headers?.get?.("X-Trace-Id") || "";
    throw e;
  }
}
