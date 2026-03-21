import code
import importlib
import typing

# SEC-001: eval
user_input = input("Enter expression: ")
result = eval(user_input)
