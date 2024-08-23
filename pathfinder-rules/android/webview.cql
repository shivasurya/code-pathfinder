/**
 * @name Android WebView JavaScript settings
 * @description Enabling JavaScript execution in a WebView can result in cross-site scripting attacks.
 * @kind problem
 * @id java/Android/webview-javascript-enabled
 * @problem.severity warning
 * @security-severity 6.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-079
 */
FIND method_invocation AS mi
WHERE mi.getName() == "setJavaScriptEnabled" && "true" in mi.getArgumentName()