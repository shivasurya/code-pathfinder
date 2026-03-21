from django.http import JsonResponse
import ast
import operator

# Safe math operations whitelist
SAFE_OPS = {
    ast.Add: operator.add,
    ast.Sub: operator.sub,
    ast.Mult: operator.mul,
    ast.Div: operator.truediv,
}

def safe_eval_math(expr):
    \"\"\"Evaluate simple math expressions safely using AST parsing.\"\"\"
    tree = ast.parse(expr, mode='eval')
    # Only allow numbers and basic arithmetic
    for node in ast.walk(tree):
        if not isinstance(node, (ast.Expression, ast.BinOp, ast.Constant,
                                 ast.Add, ast.Sub, ast.Mult, ast.Div)):
            raise ValueError("Unsafe expression")
    return eval(compile(tree, '<string>', 'eval'))

def calculate(request):
    # SECURE: Use ast.literal_eval or custom safe parser
    expression = request.GET.get('expr', '')
    try:
        result = safe_eval_math(expression)
    except (ValueError, SyntaxError):
        return JsonResponse({'error': 'Invalid expression'}, status=400)
    return JsonResponse({'result': result})

def dispatch(request):
    # SECURE: Use explicit allowlist for dispatch
    ALLOWED_ACTIONS = {
        'list': list_items,
        'search': search_items,
        'count': count_items,
    }
    action = request.GET.get('action', '')
    handler = ALLOWED_ACTIONS.get(action)
    if handler is None:
        return JsonResponse({'error': 'Unknown action'}, status=400)
    return handler(request)
