/**
 * @name unencrypted-socket
 * @description This socket is not encrypted. Use an SSLSocket created by SSLSocketFactory or SSLServerSocketFactory instead.
 * @kind problem
 * @id java/UnEncryptedSocketConnection
 * @problem.severity warning
 * @security-severity 3.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-319
 */

FROM ClassInstanceExpr AS cie
WHERE cie.getClassInstanceExpr().GetClassName() == "Socket" || cie.getClassInstanceExpr().GetClassName() == "ServerSocket"
SELECT cie.getName(), "This socket is not encrypted. Use an SSLSocket created by SSLSocketFactory or SSLServerSocketFactory instead"