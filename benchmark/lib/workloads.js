const BENCHMARK_TIMESTAMP = Date.parse('2026-01-01T00:00:00.000Z');

function buildCreateArgs(runId, index) {
  return buildEventArgs(runId, index, 'Product', 'listed', 'farmer');
}

function buildTransferArgs(runId, index) {
  return buildEventArgs(
    runId,
    index,
    'Shipment',
    'shipment_in_transit',
    'transporter',
  );
}

function buildHistoryArgs(runId, index) {
  return [`${runId}-product-${index}`];
}

function buildEventArgs(runId, index, referenceModel, eventType, actorRole) {
  const suffix = referenceModel === 'Product' ? 'create' : 'transfer';
  const referenceId =
    referenceModel === 'Product'
      ? `${runId}-product-${index}`
      : `${runId}-shipment-${index}`;
  return [
    `${runId}-${suffix}-${index}`,
    referenceId,
    referenceModel,
    `${runId}-product-${index}`,
    `${runId}-farmer-${index}`,
    eventType,
    'Lahore, Punjab',
    `${runId}-${actorRole}-${index}`,
    actorRole,
    new Date(BENCHMARK_TIMESTAMP + index * 1_000).toISOString(),
  ];
}

module.exports = {
  buildCreateArgs,
  buildHistoryArgs,
  buildTransferArgs,
};
