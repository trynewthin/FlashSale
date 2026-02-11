import http from "k6/http";
import exec from "k6/execution";
import { Rate } from "k6/metrics";

const baseUrl = (__ENV.BASE_URL || "http://127.0.0.1:8082").replace(/\/+$/, "");
const activityId = (__ENV.ACTIVITY_ID || "0").trim();
const itemId = (__ENV.ITEM_ID || "0").trim();
const duration = __ENV.TEST_DURATION || "60s";
const purchaseRate = Number(__ENV.PURCHASE_RATE || "0");
const trackRate = Number(__ENV.TRACK_RATE || "0");
const quantity = Number(__ENV.QUANTITY || "1");
const eventType = __ENV.EVENT_TYPE || "pv";
const clientIdPrefix = __ENV.CLIENT_ID_PREFIX || "k6-client";
const summaryPath = __ENV.SUMMARY_PATH || "";
const idemPrefix = __ENV.IDEM_PREFIX || "k6-open";

const purchasePreVUs = Number(__ENV.PURCHASE_PREALLOCATED_VUS || String(Math.max(20, purchaseRate * 2)));
const purchaseMaxVUs = Number(__ENV.PURCHASE_MAX_VUS || String(Math.max(purchasePreVUs, purchaseRate * 4)));
const trackPreVUs = Number(__ENV.TRACK_PREALLOCATED_VUS || String(Math.max(20, trackRate)));
const trackMaxVUs = Number(__ENV.TRACK_MAX_VUS || String(Math.max(trackPreVUs, trackRate * 2)));

const purchaseMinEffectiveRate = Number(__ENV.PURCHASE_MIN_EFFECTIVE_RATE || "0.95");
const purchaseMaxNetworkRate = Number(__ENV.PURCHASE_MAX_NETWORK_RATE || "0.03");
const purchaseP95Ms = Number(__ENV.PURCHASE_MAX_P95_MS || "5000");
const trackMinRate = Number(__ENV.TRACK_MIN_EFFECTIVE_RATE || "0.99");
const trackMaxNetworkRate = Number(__ENV.TRACK_MAX_NETWORK_RATE || "0.01");
const trackP95Ms = Number(__ENV.TRACK_MAX_P95_MS || "500");

const purchaseEffectiveRate = new Rate("purchase_effective_rate");
const purchaseOkRate = new Rate("purchase_ok_rate");
const purchaseNetworkErrorRate = new Rate("purchase_network_error_rate");
const purchaseHttp5xxRate = new Rate("purchase_http_5xx_rate");
const trackEffectiveRate = new Rate("track_effective_rate");
const trackAcceptedRate = new Rate("track_accepted_rate");
const trackNetworkErrorRate = new Rate("track_network_error_rate");
const trackHttp5xxRate = new Rate("track_http_5xx_rate");

const tokenPool = loadTokenPool(__ENV.TOKEN_FILE || "", __ENV.TOKENS || "");

if ((purchaseRate > 0 || trackRate > 0) && !isPositiveIntegerString(activityId, itemId)) {
  throw new Error("ACTIVITY_ID and ITEM_ID must be positive integer strings");
}
if (purchaseRate > 0 && tokenPool.length === 0) {
  throw new Error("purchase scenario requires TOKEN_FILE or TOKENS");
}

export const options = buildOptions();

function buildOptions() {
  const scenarios = {};
  if (purchaseRate > 0) {
    scenarios.purchase_open = {
      executor: "constant-arrival-rate",
      rate: purchaseRate,
      timeUnit: "1s",
      duration,
      preAllocatedVUs: purchasePreVUs,
      maxVUs: purchaseMaxVUs,
      exec: "purchaseOpen",
      tags: { flow: "purchase_open" },
    };
  }
  if (trackRate > 0) {
    scenarios.track_open = {
      executor: "constant-arrival-rate",
      rate: trackRate,
      timeUnit: "1s",
      duration,
      preAllocatedVUs: trackPreVUs,
      maxVUs: trackMaxVUs,
      exec: "trackOpen",
      tags: { flow: "track_open" },
    };
  }
  const thresholds = {};
  if (purchaseRate > 0) {
    thresholds.purchase_effective_rate = [`rate>=${purchaseMinEffectiveRate}`];
    thresholds.purchase_network_error_rate = [`rate<=${purchaseMaxNetworkRate}`];
    thresholds["http_req_duration{scenario:purchase_open}"] = [`p(95)<${purchaseP95Ms}`];
  }
  if (trackRate > 0) {
    thresholds.track_effective_rate = [`rate>=${trackMinRate}`];
    thresholds.track_network_error_rate = [`rate<=${trackMaxNetworkRate}`];
    thresholds["http_req_duration{scenario:track_open}"] = [`p(95)<${trackP95Ms}`];
  }
  return {
    scenarios,
    thresholds,
    summaryTrendStats: ["avg", "min", "med", "max", "p(90)", "p(95)", "p(99)"],
  };
}

function loadTokenPool(tokenFile, inlineTokens) {
  const list = [];
  if (tokenFile) {
    const content = open(tokenFile);
    content
      .split(/\r?\n/)
      .map((v) => v.trim())
      .filter((v) => v && !v.startsWith("#"))
      .forEach((v) => list.push(v));
  }
  if (inlineTokens) {
    inlineTokens
      .split(",")
      .map((v) => v.trim())
      .filter((v) => !!v)
      .forEach((v) => list.push(v));
  }
  return list;
}

function isPositiveIntegerString(...values) {
  for (const value of values) {
    if (!/^[0-9]+$/.test(value)) {
      return false;
    }
    try {
      if (BigInt(value) <= 0n) {
        return false;
      }
    } catch (_err) {
      return false;
    }
  }
  return true;
}

function parseJSONBody(res) {
  if (!res || typeof res.body !== "string" || res.body.length === 0) {
    return null;
  }
  try {
    return JSON.parse(res.body);
  } catch (_err) {
    return null;
  }
}

function isInfraFailure(res, body) {
  if (!res || res.status === 0) {
    return true;
  }
  if (res.status >= 500) {
    return true;
  }
  const code = body && body.code ? String(body.code) : "";
  return code === "SYS_INTERNAL" || code === "DB_ERROR";
}

export function purchaseOpen() {
  const token = tokenPool[exec.scenario.iterationInTest % tokenPool.length];
  const idem = `${idemPrefix}-${Date.now()}-${exec.vu.idInTest}-${exec.scenario.iterationInTest}`;
  // int64 ID 可能超过 JS 安全整数，手工拼接 JSON 保持精度。
  const payload = `{"activity_item_id":${itemId},"quantity":${quantity},"idempotency_key":"${idem}"}`;
  const res = http.post(`${baseUrl}/api/v1/seckill/activities/${activityId}/purchase`, payload, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    tags: { flow: "purchase_open" },
  });
  const body = parseJSONBody(res);
  const networkErr = !res || res.status === 0;
  const http5xx = !!res && res.status >= 500;
  const infraFail = isInfraFailure(res, body);
  const code = body && body.code ? String(body.code) : "";
  const ok = !!res && res.status === 200 && code === "OK";

  purchaseNetworkErrorRate.add(networkErr);
  purchaseHttp5xxRate.add(http5xx);
  purchaseEffectiveRate.add(!infraFail);
  purchaseOkRate.add(ok);
}

export function trackOpen() {
  const idem = `${idemPrefix}-track-${Date.now()}-${exec.vu.idInTest}-${exec.scenario.iterationInTest}`;
  const clientId = `${clientIdPrefix}-${exec.vu.idInTest}-${exec.scenario.iterationInTest}`;
  const occurredAt = Math.floor(Date.now() / 1000);
  const payload = `{"activity_item_id":${itemId},"event_type":"${eventType}","client_id":"${clientId}","idempotency_key":"${idem}","occurred_at_unix":${occurredAt}}`;
  const res = http.post(`${baseUrl}/api/v1/seckill/activities/${activityId}/track`, payload, {
    headers: { "Content-Type": "application/json" },
    tags: { flow: "track_open" },
  });
  const body = parseJSONBody(res);
  const networkErr = !res || res.status === 0;
  const http5xx = !!res && res.status >= 500;
  const infraFail = isInfraFailure(res, body);
  const accepted = !!res && res.status === 200 && body && body.code === "OK" && body.data && body.data.accepted === true;

  trackNetworkErrorRate.add(networkErr);
  trackHttp5xxRate.add(http5xx);
  trackEffectiveRate.add(!infraFail);
  trackAcceptedRate.add(accepted);
}

function getMetricValue(data, name, key, fallback = 0) {
  const metric = data.metrics && data.metrics[name];
  if (!metric || !metric.values) {
    return fallback;
  }
  const v = metric.values[key];
  return typeof v === "number" ? v : fallback;
}

function renderSummary(data) {
  const lines = [];
  lines.push("=== k6 seckill open-model summary ===");
  lines.push(`generated_at=${new Date().toISOString()}`);
  lines.push(`base_url=${baseUrl}`);
  lines.push(`activity_id=${activityId} item_id=${itemId}`);
  lines.push(`purchase_rate=${purchaseRate} track_rate=${trackRate} duration=${duration}`);
  lines.push(`http_req_duration_p95=${getMetricValue(data, "http_req_duration", "p(95)", 0).toFixed(2)}ms`);
  lines.push(`purchase_effective_rate=${getMetricValue(data, "purchase_effective_rate", "rate", 0).toFixed(4)}`);
  lines.push(`purchase_ok_rate=${getMetricValue(data, "purchase_ok_rate", "rate", 0).toFixed(4)}`);
  lines.push(`purchase_network_error_rate=${getMetricValue(data, "purchase_network_error_rate", "rate", 0).toFixed(4)}`);
  lines.push(`track_effective_rate=${getMetricValue(data, "track_effective_rate", "rate", 0).toFixed(4)}`);
  lines.push(`track_accepted_rate=${getMetricValue(data, "track_accepted_rate", "rate", 0).toFixed(4)}`);
  lines.push(`track_network_error_rate=${getMetricValue(data, "track_network_error_rate", "rate", 0).toFixed(4)}`);
  return `${lines.join("\n")}\n`;
}

export function handleSummary(data) {
  const summary = {
    generated_at: new Date().toISOString(),
    config: {
      base_url: baseUrl,
      activity_id: activityId,
      item_id: itemId,
      duration,
      purchase_rate: purchaseRate,
      track_rate: trackRate,
      purchase_preallocated_vus: purchasePreVUs,
      purchase_max_vus: purchaseMaxVUs,
      track_preallocated_vus: trackPreVUs,
      track_max_vus: trackMaxVUs,
    },
    result: {
      http_req_duration_p95: getMetricValue(data, "http_req_duration", "p(95)", 0),
      http_req_duration_p99: getMetricValue(data, "http_req_duration", "p(99)", 0),
      purchase_effective_rate: getMetricValue(data, "purchase_effective_rate", "rate", 0),
      purchase_ok_rate: getMetricValue(data, "purchase_ok_rate", "rate", 0),
      purchase_network_error_rate: getMetricValue(data, "purchase_network_error_rate", "rate", 0),
      purchase_http_5xx_rate: getMetricValue(data, "purchase_http_5xx_rate", "rate", 0),
      track_effective_rate: getMetricValue(data, "track_effective_rate", "rate", 0),
      track_accepted_rate: getMetricValue(data, "track_accepted_rate", "rate", 0),
      track_network_error_rate: getMetricValue(data, "track_network_error_rate", "rate", 0),
      track_http_5xx_rate: getMetricValue(data, "track_http_5xx_rate", "rate", 0),
    },
    metrics: data.metrics,
  };

  const outputs = {
    stdout: renderSummary(data),
  };
  if (summaryPath) {
    outputs[summaryPath] = JSON.stringify(summary, null, 2);
  }
  return outputs;
}
