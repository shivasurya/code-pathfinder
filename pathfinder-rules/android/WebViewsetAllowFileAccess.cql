/**
 * @name Android WebView JavaScript settings
 * @description Enabling setAllowFileAccess enables webview access to file:/// URLs.
 * @kind problem
 * @id java/Android/webview-javascript-enabled
 * @problem.severity warning
 * @security-severity 6.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-079
 */
FROM method_invocation AS mi
WHERE mi.getName() == "setAllowFileAccess" && "true" in mi.getArgumentName()