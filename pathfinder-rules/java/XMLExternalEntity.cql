/**
 * @name XXEConfig
 * @description Detects insecure XML parsers and configurations that could lead to XXE attacks
 * @kind problem
 * @id java/XXEConfig
 * @problem.severity warning
 * @security-severity 8.0
 * @precision high
 * @tags security
 * external/cwe/cwe-611
 * @ruleprovider java
 */

FROM method_invocation AS mi
WHERE mi.getName() == "setFeature" &&
    ("http://xml.org/sax/features/external-parameter-entities" in mi.getArgumentName() &&
     "true" in mi.getArgumentName()) ||
    ("http://xml.org/sax/features/external-general-entities" in mi.getArgumentName() && 
     "true" in mi.getArgumentName()) ||
    ("http://apache.org/xml/features/disallow-doctype-decl" in mi.getArgumentName() &&
     "false" in mi.getArgumentName())
SELECT mi.getName(), "XML External Entity (XXE) attack vulnerability"
