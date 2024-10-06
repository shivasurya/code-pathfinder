/**
 * @name DefaultHttpClient
 * @description The DefaultHttpClient is deprecated. Use HttpClientBuilder instead.
 * @kind problem
 * @id java/DefaultHttpClient
 * @problem.severity warning
 * @security-severity 3.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-326
 */

 FROM ClassInstanceExpr AS cie
 WHERE cie.getClassInstanceExpr().GetClassName() == "DefaultHttpClient"
 SELECT cie.getName(), "The DefaultHttpClient is deprecated. Use HttpClientBuilder instead."
