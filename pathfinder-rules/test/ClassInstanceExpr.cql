FROM ClassInstanceExpr AS cie
WHERE cie.getClassInstanceExpr().GetClassName() == "Intent"
SELECT cie.getClassInstanceExpr().GetArgs(), cie.getClassInstanceExpr().GetNumArgs(), cie.getClassInstanceExpr().GetArg(0)