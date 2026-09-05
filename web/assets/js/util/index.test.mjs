import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";

const source = await readFile(new URL("./index.js", import.meta.url), "utf8");

function loadHttpUtil(calls) {
  const context = {
    console,
    axios: {
      put: async (url, data, options) => {
        calls.push({ method: "put", url, data, options });
        return { data: { success: true, msg: "", obj: null } };
      },
      delete: async (url, options) => {
        calls.push({ method: "delete", url, options });
        return { data: { success: true, msg: "", obj: null } };
      },
    },
    Vue: { prototype: { $message: { success() {}, error() {} } } },
  };
  vm.createContext(context);
  vm.runInContext(source, context);
  return vm.runInContext("HttpUtil", context);
}

test("HttpUtil.put forwards URL, payload, and options", async () => {
  const calls = [];
  const HttpUtil = loadHttpUtil(calls);
  assert.equal(typeof HttpUtil.put, "function");
  await HttpUtil.put("/panel/packages/7", { name: "Starter" }, { timeout: 1000 });
  assert.deepEqual(calls, [{ method: "put", url: "/panel/packages/7", data: { name: "Starter" }, options: { timeout: 1000 } }]);
});

test("HttpUtil.delete forwards URL and options", async () => {
  const calls = [];
  const HttpUtil = loadHttpUtil(calls);
  assert.equal(typeof HttpUtil.delete, "function");
  await HttpUtil.delete("/panel/packages/7", { timeout: 1000 });
  assert.deepEqual(calls, [{ method: "delete", url: "/panel/packages/7", options: { timeout: 1000 } }]);
});
