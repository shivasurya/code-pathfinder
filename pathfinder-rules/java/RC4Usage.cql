/**
 * @name RC4Usage
 * @description RC4/RC2 is insecure. Use an alternative cipher suite.
 * @kind problem
 * @id java/RC4Usage
 * @problem.severity warning
 * @security-severity 3.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-327
 * @ruleprovider java
 */

FROM method_invocation AS mi
WHERE mi.getName() == "Cipher.getInstance"
&& ("RC4" in mi.getArgumentName() || "RC2" in mi.getArgumentName())
SELECT mi.getName(), "RC4/RC2 is insecure. Use an alternative cipher suite."