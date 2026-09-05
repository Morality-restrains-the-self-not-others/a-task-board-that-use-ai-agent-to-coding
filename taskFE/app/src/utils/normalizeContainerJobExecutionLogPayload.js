/**
 * Normalize GET container-job-execution-log JSON to the SaaS zTree contract:
 * `{ job, steps, layer_changes }`.
 *
 * taskContainerGateway briefly returned a flattened job (top-level id/output/status
 * + steps as an array). The UI reads `payload.job` / `payload.steps.steps`, so a
 * flat body renders as 「暂无任务日志」 while the container page (direct /api/jobs)
 * still shows logs.
 *
 * @param {unknown} body
 * @returns {Record<string, unknown> | null}
 */
export function normalizeContainerJobExecutionLogPayload(body) {
  if (!body || typeof body !== 'object' || Array.isArray(body)) return null
  const src = /** @type {Record<string, unknown>} */ (body)

  if (src.job && typeof src.job === 'object' && !Array.isArray(src.job)) {
    const stepsRaw = src.steps
    const steps =
      stepsRaw && typeof stepsRaw === 'object' && !Array.isArray(stepsRaw)
        ? stepsRaw
        : Array.isArray(stepsRaw)
          ? { steps: stepsRaw }
          : { steps: [] }
    return {
      ...src,
      job: src.job,
      steps,
      layer_changes:
        src.layer_changes && typeof src.layer_changes === 'object' && !Array.isArray(src.layer_changes)
          ? src.layer_changes
          : src.layer_changes ?? null,
    }
  }

  // Flat gateway bug: job fields at top level
  const { steps: flatSteps, layer_changes: flatLc, ...jobFields } = src
  if (
    jobFields.id == null &&
    jobFields.output == null &&
    jobFields.status == null &&
    jobFields.command == null
  ) {
    return src
  }
  const steps =
    flatSteps && typeof flatSteps === 'object' && !Array.isArray(flatSteps)
      ? flatSteps
      : { steps: Array.isArray(flatSteps) ? flatSteps : [] }
  return {
    job: jobFields,
    steps,
    layer_changes:
      flatLc && typeof flatLc === 'object' && !Array.isArray(flatLc) ? flatLc : null,
  }
}
