"""
Taint propagation primitives for dataflow analysis.

These primitives define HOW taint propagates through code constructs.
Developers specify which primitives to enable via propagates_through parameter.
"""

from typing import Dict, Any, List, Optional
from enum import Enum


class PropagationType(Enum):
    """
    Enum of all propagation primitive types.

    Phase 1 (MVP - This PR):
        ASSIGNMENT, FUNCTION_ARGS, FUNCTION_RETURNS

    Phase 2 (MVP - Future PR):
        STRING_CONCAT, STRING_FORMAT

    Phase 3-6 (Post-MVP):
        Collections, control flow, OOP, advanced
    """

    # ===== PHASE 1: BARE MINIMUM (MVP) =====
    ASSIGNMENT = "assignment"
    FUNCTION_ARGS = "function_args"
    FUNCTION_RETURNS = "function_returns"

    # ===== PHASE 2: STRING OPERATIONS (MVP - Future PR) =====
    STRING_CONCAT = "string_concat"
    STRING_FORMAT = "string_format"

    # ===== PHASE 3: COLLECTIONS (POST-MVP) =====
    LIST_APPEND = "list_append"
    LIST_EXTEND = "list_extend"
    DICT_VALUES = "dict_values"
    DICT_UPDATE = "dict_update"
    SET_ADD = "set_add"

    # ===== PHASE 4: CONTROL FLOW (POST-MVP) =====
    IF_CONDITION = "if_condition"
    FOR_ITERATION = "for_iteration"
    WHILE_CONDITION = "while_condition"
    SWITCH_CASE = "switch_case"

    # ===== PHASE 5: OOP (POST-MVP) =====
    ATTRIBUTE_ASSIGNMENT = "attribute_assignment"
    METHOD_CALL = "method_call"
    CONSTRUCTOR = "constructor"

    # ===== PHASE 6: ADVANCED (POST-MVP) =====
    COMPREHENSION = "comprehension"
    LAMBDA_CAPTURE = "lambda_capture"
    YIELD_STMT = "yield_stmt"


class PropagationPrimitive:
    """
    Base class for propagation primitives.

    Each primitive describes ONE way taint can flow through code.
    """

    def __init__(
        self, prim_type: PropagationType, metadata: Optional[Dict[str, Any]] = None
    ):
        """
        Args:
            prim_type: The type of propagation
            metadata: Optional additional configuration
        """
        self.type = prim_type
        self.metadata = metadata or {}

    def to_ir(self) -> Dict[str, Any]:
        """
        Serialize to JSON IR.

        Returns:
            {
                "type": "assignment",
                "metadata": {}
            }
        """
        return {
            "type": self.type.value,
            "metadata": self.metadata,
        }

    def __repr__(self) -> str:
        return f"propagates.{self.type.value}()"


class propagates:
    """
    Namespace for taint propagation primitives.

    Usage:
        propagates.assignment()
        propagates.function_args()
        propagates.function_returns()
    """

    # ===== PHASE 1: BARE MINIMUM (MVP - THIS PR) =====

    @staticmethod
    def assignment() -> PropagationPrimitive:
        """
        Taint propagates through variable assignment.

        Patterns matched:
            x = tainted           # Simple assignment
            a = b = tainted       # Chained assignment
            x, y = tainted, safe  # Tuple unpacking (x is tainted)

        This is the MOST COMMON propagation pattern (~40% of all flows).

        Examples:
            user_input = request.GET.get("id")  # source
            query = user_input                  # PROPAGATES via assignment
            cursor.execute(query)               # sink

        Returns:
            PropagationPrimitive for assignment
        """
        return PropagationPrimitive(PropagationType.ASSIGNMENT)

    @staticmethod
    def function_args() -> PropagationPrimitive:
        """
        Taint propagates through function arguments.

        Patterns matched:
            func(tainted)              # Positional argument
            func(arg=tainted)          # Keyword argument
            func(*tainted)             # Args unpacking
            func(**tainted)            # Kwargs unpacking

        Critical for inter-procedural analysis (~30% of flows).

        Examples:
            user_input = request.GET.get("id")  # source
            process_data(user_input)            # PROPAGATES via function_args
            def process_data(data):
                execute(data)                   # sink (data is tainted)

        Returns:
            PropagationPrimitive for function arguments
        """
        return PropagationPrimitive(PropagationType.FUNCTION_ARGS)

    @staticmethod
    def function_returns() -> PropagationPrimitive:
        """
        Taint propagates through return values.

        Patterns matched:
            return tainted                # Direct return
            return tainted if cond else safe  # Conditional return
            return [tainted, safe]        # Return list containing tainted

        Essential for functions that transform tainted data (~20% of flows).

        Examples:
            def get_user_id():
                user_input = request.GET.get("id")  # source
                return user_input                   # PROPAGATES via return

            query = get_user_id()           # query is now tainted
            execute(query)                  # sink

        Returns:
            PropagationPrimitive for function returns
        """
        return PropagationPrimitive(PropagationType.FUNCTION_RETURNS)

    # ===== PHASE 2: STRING OPERATIONS (MVP - THIS PR) =====

    @staticmethod
    def string_concat() -> PropagationPrimitive:
        """
        Taint propagates through string concatenation.

        Patterns matched:
            result = tainted + "suffix"      # Right concat
            result = "prefix" + tainted      # Left concat
            result = tainted + safe + more   # Mixed concat

        Critical for SQL/Command injection where queries are built via concat (~10% of flows).

        Examples:
            user_id = request.GET.get("id")           # source
            query = "SELECT * FROM users WHERE id = " + user_id  # PROPAGATES via string_concat
            cursor.execute(query)                     # sink

        Returns:
            PropagationPrimitive for string concatenation
        """
        return PropagationPrimitive(PropagationType.STRING_CONCAT)

    @staticmethod
    def string_format() -> PropagationPrimitive:
        """
        Taint propagates through string formatting.

        Patterns matched:
            f"{tainted}"                    # f-string
            "{}".format(tainted)            # str.format()
            "%s" % tainted                  # % formatting
            "{name}".format(name=tainted)   # Named placeholders

        Critical for SQL injection where ORM methods use format() (~8% of flows).

        Examples:
            user_id = request.GET.get("id")           # source
            query = f"SELECT * FROM users WHERE id = {user_id}"  # PROPAGATES via string_format
            cursor.execute(query)                     # sink

        Returns:
            PropagationPrimitive for string formatting
        """
        return PropagationPrimitive(PropagationType.STRING_FORMAT)

    # ===== PHASE 3-6: POST-MVP =====
    # Will be implemented in post-MVP PRs


def create_propagation_list(
    primitives: List[PropagationPrimitive],
) -> List[Dict[str, Any]]:
    """
    Convert a list of propagation primitives to JSON IR.

    Args:
        primitives: List of PropagationPrimitive objects

    Returns:
        List of JSON IR dictionaries

    Example:
        >>> prims = [propagates.assignment(), propagates.function_args()]
        >>> create_propagation_list(prims)
        [
            {"type": "assignment", "metadata": {}},
            {"type": "function_args", "metadata": {}}
        ]
    """
    return [prim.to_ir() for prim in primitives]
