// source: https://github.com/benjamingr/RegExp.escape/blob/e039c25e5d88bf7b567c1be9a6e7bf16a126a70e/polyfill.js#L1-L6
if (!RegExp.escape) {
  RegExp.escape = function (s) {
    return String(s).replace(/[\\^$*+?.()|[\]{}]/g, '\\$&');
  };
}
