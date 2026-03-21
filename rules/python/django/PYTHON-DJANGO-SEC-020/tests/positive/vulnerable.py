

# SEC-020: eval with request data
def vulnerable_eval(request):
    expr = request.GET.get('expr')
    result = eval(expr)
    return result


    code = request.POST.get('code')
    exec(code)


    func_name = request.GET.get('func')
    func = globals().get(func_name)
    return func()
