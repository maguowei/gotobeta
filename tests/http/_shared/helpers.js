export function createUniqueSuffix(prefix = "case") {
  const random = Math.floor(Math.random() * 10000).toString().padStart(4, "0");
  return `${prefix}-${Date.now()}-${random}`;
}

export function assertStatus(response, expected) {
  client.test(`HTTP ${expected}`, () => {
    client.assert(
      response.status === expected,
      `期望 ${expected}，实际 ${response.status}`
    );
  });
}

export function assertBodyField(response, field, expected) {
  client.test(`body.${field} === ${expected}`, () => {
    client.assert(
      response.body[field] === expected,
      `期望 ${field}=${expected}，实际 ${response.body[field]}`
    );
  });
}
