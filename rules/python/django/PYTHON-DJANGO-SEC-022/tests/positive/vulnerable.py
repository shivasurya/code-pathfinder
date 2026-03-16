

    expr = request.GET.get('expr')
    result = eval(expr)
    return result


    code = request.POST.get('code')
    exec(code)


# SEC-022: globals misuse
def vulnerable_globals(request):
    func_name = request.GET.get('func')
    func = globals().get(func_name)
    return func()
