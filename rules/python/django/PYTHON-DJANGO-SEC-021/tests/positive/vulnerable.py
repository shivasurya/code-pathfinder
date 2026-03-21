

    expr = request.GET.get('expr')
    result = eval(expr)
    return result


# SEC-021: exec with request data
def vulnerable_exec(request):
    code = request.POST.get('code')
    exec(code)


    func_name = request.GET.get('func')
    func = globals().get(func_name)
    return func()
