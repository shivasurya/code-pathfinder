/**
 * @name BlowfishUsage
 * @description Use of Blowfish was detected. Blowfish uses a 64-bit block size
 * @kind problem
 * @id java/BlowfishUsage
 * @problem.severity warning
 * @security-severity 3.1
 * @precision medium
 * @tags security
 * external/cwe/cwe-327
 * @ruleprovider java
 */

FROM method_invocation AS mi
WHERE mi.getName() == "Cipher.getInstance"
&& "Blowfish" in mi.getArgumentName()
SELECT mi.getName(), "Use of Blowfish was detected. Blowfish uses a 64-bit block size
    that  makes it vulnerable to birthday attacks, and is therefore considered
      non-compliant."