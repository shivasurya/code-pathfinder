/**
 * @name deprecated-httpclient
 * @description The DefaultHttpClient is deprecated. Use HttpClientBuilder instead.
 * @kind problem
 * @id java/deprecated-httpclient
 * @problem.severity warning
 * @security-severity 2
 * @precision high
 * @tags security
 * external/cwe/cwe-326
 */
 FROM ClassInstanceExpr AS cie
 WHERE cie.getClassInstanceExpr().GetClassName() == "DefaultHttpClient"
 SELECT cie.getName(), "The DefaultHttpClient is deprecated. Use HttpClientBuilder instead."
