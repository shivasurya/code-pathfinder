/**
 * @name SHA1Usage
 * @description SHA1 is not collision resistant and is therefore not suitable as a cryptographic signature
 * @kind problem
 * @id java/SHA1Usage
 * @problem.severity warning
 * @security-severity 3.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-328
 */

FROM method_invocation AS mi
WHERE mi.getName() == "MessageDigest.getInstance"
&& ("SHA1" in mi.getArgumentName() || "SHA-1" in mi.getArgumentName())
SELECT mi.getName(), "SHA1 is not collision resistant and is therefore not suitable as a cryptographic signature"