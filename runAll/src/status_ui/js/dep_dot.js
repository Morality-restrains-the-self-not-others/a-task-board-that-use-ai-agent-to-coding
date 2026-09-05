// Dependency-row dots must follow the live service in the same /api/status
// payload. depends_on.status can be stale after config reload (pending/empty)
// even when the named dependency is already healthy.

function indexServicesByName(services) {
  const map = new Map();
  if (!Array.isArray(services)) {
    return map;
  }
  for (const svc of services) {
    if (svc && typeof svc.name === 'string' && svc.name) {
      map.set(svc.name, svc);
    }
  }
  return map;
}

function resolveDepDotClass(dep, liveSvc, classOf) {
  const classify = typeof classOf === 'function' ? classOf : function () { return 'gray'; };
  if (liveSvc) {
    return classify(liveSvc);
  }
  if (!dep) {
    return classify(null);
  }
  return classify({ status: dep.status });
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    indexServicesByName: indexServicesByName,
    resolveDepDotClass: resolveDepDotClass
  };
}
