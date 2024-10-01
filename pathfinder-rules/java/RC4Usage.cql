FROM method_invocation AS mi
WHERE mi.getName() == "Cipher.getInstance"
&& ("RC4" in mi.getArgumentName() || "RC2" in mi.getArgumentName())
SELECT mi.getName(), "RC4/RC2 is insecure. Use an alternative cipher suite."