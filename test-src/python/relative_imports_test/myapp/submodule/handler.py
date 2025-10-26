# Test file with relative imports from submodule
# This file is at: myapp.submodule.handler

# Single dot - import from current package (myapp.submodule)
from . import utils

# Two dots - import from parent package (myapp)
from .. import config

# Two dots with submodule - import from parent's sibling (myapp.utils)
from ..utils import helper

# Two dots with another submodule - import from parent's sibling (myapp.config)
from ..config import settings
