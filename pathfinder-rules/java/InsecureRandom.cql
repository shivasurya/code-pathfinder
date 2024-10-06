/**
 * @name InsecureRandom
 * @description Math.random() is not cryptographically secure. Use SecureRandom instead.
 * @kind problem
 * @id java/InsecureRandom
 * @problem.severity warning
 * @security-severity 3.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-330
 */

FROM method_invocation AS mi
WHERE mi.getName() == "Math.random"
SELECT mi.getName(), "Math.random() is not cryptographically secure. Use SecureRandom instead."