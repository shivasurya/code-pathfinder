/**
 * @name Android WebView JavaScript settings
 * @description Enabling setAllowFileAccessFromFileURLs leak s&&box access to file:/// URLs.
 * @kind problem
 * @id java/Android/webview-javascript-enabled
 * @problem.severity warning
 * @security-severity 6.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-079
 */
FROM method_invocation AS mi
WHERE mi.getName() == "setAllowFileAccessFromFileURLs" && "true" in mi.getArgumentName()
SELECT mi.getName(), "File access enabled"