FROM method_invocation AS mi
WHERE mi.getName() == "Math.random"
SELECT mi.getName(), "Math.random() is not cryptographically secure. Use SecureRandom instead."