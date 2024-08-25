/**
 * @name Android WebView JavaScript settings
 * @description Enabling setAllowContentAccess enables content:// access from webpages.
 * @kind problem
 * @id java/Android/webview-javascript-enabled
 * @problem.severity warning
 * @security-severity 6.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-079
 */
FIND method_invocation AS mi
WHERE mi.getName() == "setAllowContentAccess" && "true" in mi.getArgumentName()