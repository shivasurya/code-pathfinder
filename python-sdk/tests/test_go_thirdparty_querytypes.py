"""Tests for Go third-party framework QueryType classes (PR-04)."""

import pytest
from codepathfinder.go_rule import (
    GoGormDB,
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    GoGRPCServerTransportStream,
    GoPgxConn,
    GoSqlxDB,
    GoRedisClient,
    GoMongoCollection,
    GoJWTToken,
    GoGorillaMuxRouter,
    GoRestyClient,
    GoChiRouter,
    GoViperConfig,
    GoYAMLDecoder,
)

_ALL_THIRD_PARTY_TYPES = [
    GoGormDB,
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    GoGRPCServerTransportStream,
    GoPgxConn,
    GoSqlxDB,
    GoRedisClient,
    GoMongoCollection,
    GoJWTToken,
    GoGorillaMuxRouter,
    GoRestyClient,
    GoChiRouter,
    GoViperConfig,
    GoYAMLDecoder,
]


class TestGoThirdPartyQueryTypes:

    def test_all_querytypes_importable(self):
        """All 15 QueryType classes must be importable from go_rule."""
        assert len(_ALL_THIRD_PARTY_TYPES) == 15

    def test_all_querytypes_have_fqns(self):
        """Every QueryType must have at least one FQN."""
        for qt in _ALL_THIRD_PARTY_TYPES:
            assert len(qt.fqns) >= 1, f"{qt.__name__} has no FQNs"

    def test_fqns_are_strings(self):
        """All FQNs must be strings."""
        for qt in _ALL_THIRD_PARTY_TYPES:
            for fqn in qt.fqns:
                assert isinstance(fqn, str), f"{qt.__name__} FQN {fqn!r} is not a string"

    def test_fqns_are_fully_qualified(self):
        """All FQNs must contain a dot (package.Type format)."""
        for qt in _ALL_THIRD_PARTY_TYPES:
            for fqn in qt.fqns:
                assert "." in fqn, f"{qt.__name__} FQN {fqn!r} is not fully qualified"

    # --- Individual FQN spot checks ---

    def test_gorm_db_fqns(self):
        assert "gorm.io/gorm.DB" in GoGormDB.fqns

    def test_gin_context_fqns(self):
        assert "github.com/gin-gonic/gin.Context" in GoGinContext.fqns

    def test_echo_context_fqns(self):
        assert "github.com/labstack/echo/v4.Context" in GoEchoContext.fqns

    def test_fiber_ctx_fqns(self):
        assert "github.com/gofiber/fiber/v2.Ctx" in GoFiberCtx.fqns

    def test_grpc_stream_fqns(self):
        assert "google.golang.org/grpc.ServerTransportStream" in GoGRPCServerTransportStream.fqns

    def test_redis_client_fqns(self):
        assert "github.com/redis/go-redis/v9.Client" in GoRedisClient.fqns

    def test_jwt_token_fqns(self):
        assert "github.com/golang-jwt/jwt/v5.Token" in GoJWTToken.fqns

    def test_gorilla_mux_fqns(self):
        assert "github.com/gorilla/mux.Router" in GoGorillaMuxRouter.fqns

    def test_viper_config_fqns(self):
        assert "github.com/spf13/viper.Viper" in GoViperConfig.fqns

    def test_yaml_decoder_fqns(self):
        assert "gopkg.in/yaml.v3.Decoder" in GoYAMLDecoder.fqns

    # --- Multi-FQN types ---

    def test_pgx_conn_has_two_fqns(self):
        assert len(GoPgxConn.fqns) == 2
        assert "github.com/jackc/pgx/v5.Conn" in GoPgxConn.fqns
        assert "github.com/jackc/pgx/v5/pgxpool.Pool" in GoPgxConn.fqns

    def test_sqlx_db_has_two_fqns(self):
        assert len(GoSqlxDB.fqns) == 2
        assert "github.com/jmoiron/sqlx.DB" in GoSqlxDB.fqns
        assert "github.com/jmoiron/sqlx.Tx" in GoSqlxDB.fqns

    def test_mongo_collection_has_two_fqns(self):
        assert len(GoMongoCollection.fqns) == 2
        assert "go.mongodb.org/mongo-driver/mongo.Collection" in GoMongoCollection.fqns
        assert "go.mongodb.org/mongo-driver/mongo.Client" in GoMongoCollection.fqns

    def test_resty_client_has_two_fqns(self):
        assert len(GoRestyClient.fqns) == 2
        assert "github.com/go-resty/resty/v2.Client" in GoRestyClient.fqns
        assert "github.com/go-resty/resty/v2.Request" in GoRestyClient.fqns

    def test_chi_router_has_two_fqns(self):
        assert len(GoChiRouter.fqns) == 2
        assert "github.com/go-chi/chi/v5.Router" in GoChiRouter.fqns
        assert "github.com/go-chi/chi/v5.Mux" in GoChiRouter.fqns

    # --- MethodMatcher IR verification ---

    def test_gorm_method_matcher_ir(self):
        """GoGormDB.method() produces correct IR with GORM FQN."""
        matcher = GoGormDB.method("Raw", "Exec")
        ir = matcher.to_ir()
        assert ir["type"] == "type_constrained_call"
        assert "gorm.io/gorm.DB" in ir["receiverTypes"]
        assert "Raw" in ir["methodNames"]
        assert "Exec" in ir["methodNames"]

    def test_gin_context_method_matcher_ir(self):
        """GoGinContext.method() produces correct IR with Gin FQN."""
        matcher = GoGinContext.method("Query", "Param")
        ir = matcher.to_ir()
        assert ir["type"] == "type_constrained_call"
        assert "github.com/gin-gonic/gin.Context" in ir["receiverTypes"]
        assert "Query" in ir["methodNames"]
        assert "Param" in ir["methodNames"]

    def test_gorm_matcher_excludes_stdlib_sql(self):
        """GoGormDB.method() receiverTypes must NOT include database/sql.DB."""
        matcher = GoGormDB.method("Raw")
        ir = matcher.to_ir()
        assert "database/sql.DB" not in ir["receiverTypes"]

    def test_gin_matcher_excludes_echo(self):
        """GoGinContext.method() receiverTypes must NOT include echo.Context."""
        matcher = GoGinContext.method("Query")
        ir = matcher.to_ir()
        assert "github.com/labstack/echo/v4.Context" not in ir["receiverTypes"]

    def test_pgx_conn_matcher_includes_both_fqns(self):
        """GoPgxConn.method() must include both Conn and Pool FQNs."""
        matcher = GoPgxConn.method("Query")
        ir = matcher.to_ir()
        assert "github.com/jackc/pgx/v5.Conn" in ir["receiverTypes"]
        assert "github.com/jackc/pgx/v5/pgxpool.Pool" in ir["receiverTypes"]

    def test_resty_client_matcher_includes_both_fqns(self):
        """GoRestyClient.method() must include both Client and Request FQNs."""
        matcher = GoRestyClient.method("Get", "Post")
        ir = matcher.to_ir()
        assert "github.com/go-resty/resty/v2.Client" in ir["receiverTypes"]
        assert "github.com/go-resty/resty/v2.Request" in ir["receiverTypes"]

    def test_method_requires_at_least_one_name(self):
        """Calling .method() with no arguments must raise ValueError."""
        with pytest.raises(ValueError):
            GoGormDB.method()

    def test_no_duplicate_fqns_across_types(self):
        """No FQN should appear in more than one QueryType class."""
        seen: dict[str, str] = {}
        for qt in _ALL_THIRD_PARTY_TYPES:
            for fqn in qt.fqns:
                assert fqn not in seen, (
                    f"Duplicate FQN {fqn!r} in {qt.__name__} and {seen[fqn]}"
                )
                seen[fqn] = qt.__name__
