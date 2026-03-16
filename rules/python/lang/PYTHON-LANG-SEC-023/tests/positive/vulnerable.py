import subprocess
import asyncio

# SEC-023: subinterpreters
import _xxsubinterpreters
_xxsubinterpreters.run_string("print('hello')")
