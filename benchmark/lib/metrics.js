function summarize(latenciesMs, startedAtMs, completedAtMs) {
  if (!Array.isArray(latenciesMs) || latenciesMs.length === 0) {
    throw new Error('at least one successful measurement is required');
  }
  if (!Number.isFinite(startedAtMs) || !Number.isFinite(completedAtMs)) {
    throw new Error('workload timestamps must be finite');
  }

  const elapsedMs = completedAtMs - startedAtMs;
  if (elapsedMs <= 0) {
    throw new Error('workload completion time must be after start time');
  }

  const sorted = [...latenciesMs].sort((left, right) => left - right);
  return {
    count: sorted.length,
    minMs: sorted[0],
    p50Ms: percentile(sorted, 0.5),
    p95Ms: percentile(sorted, 0.95),
    maxMs: sorted.at(-1),
    elapsedMs,
    throughputTps: (sorted.length * 1_000) / elapsedMs,
  };
}

function percentile(sorted, fraction) {
  return sorted[Math.ceil(sorted.length * fraction) - 1];
}

module.exports = { summarize };
