/**
 * @name Android WebView JavaScript settings
 * @description Enabling addJavascriptInterface exposes java methods to JavaScript.
 * @kind problem
 * @id java/Android/webview-javascript-enabled
 * @problem.severity warning
 * @security-severity 8.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-079
 */

predicate isJavaScriptEnabled(method_invocation mi) {
  mi.getName() == "addJavascriptInterface"
}

FROM method_invocation AS mi
WHERE isJavaScriptEnabled(mi)
SELECT mi.getName(), "JavaScript interface exposed"