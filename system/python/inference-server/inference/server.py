import logging
import threading
from concurrent import futures
from datetime import UTC, datetime, timedelta

import grpc
from google.protobuf.timestamp_pb2 import Timestamp
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from weather.inference.v1 import inference_pb2, inference_pb2_grpc

from inference.backends import create_backend
from inference.backends.base import ForecastBackend
from inference.config import Config
from inference.features import (
    MAX_GAP_DAILY,
    MAX_GAP_HOURLY,
    calculate_temporal_covariates,
    extract_observation_dataframe,
    filter_and_impute_columns,
    floor_to_hour,
    generate_daily_future_timestamps,
    generate_hourly_future_timestamps,
    sort_observations,
)
from inference.formatter import format_daily_predictions, format_hourly_predictions
from inference.validator import (
    ValidationError,
    validate_daily_request,
    validate_hourly_request,
)

logger = logging.getLogger(__name__)


class InferenceServicer(inference_pb2_grpc.InferenceServiceServicer):
    """InferenceServicer handles weather forecasting and model metadata gRPC requests."""

    def __init__(self, backend: ForecastBackend, config: Config):
        """InferenceServicer stores the forecast backend and configuration."""
        self.backend = backend
        self.config = config

    def PredictHourlyWeather(
        self,
        request: inference_pb2.PredictHourlyWeatherRequest,
        context: grpc.ServicerContext,
    ) -> inference_pb2.PredictHourlyWeatherResponse:
        """PredictHourlyWeather runs ingress validation, feature transformations, and hourly forecasting."""
        try:
            validate_hourly_request(request)
        except ValidationError as validation_err:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(validation_err))
            return inference_pb2.PredictHourlyWeatherResponse()

        prediction_length = request.horizon_hours
        max_gap = MAX_GAP_HOURLY

        issue_time_dt = floor_to_hour(request.issue_time.ToDatetime().replace(tzinfo=UTC))
        aligned_issue_time = Timestamp()
        aligned_issue_time.FromDatetime(issue_time_dt)

        sorted_observations = sort_observations(request.history)
        observation_df = extract_observation_dataframe(
            sorted_observations,
            step=timedelta(hours=1),
            issue_time=issue_time_dt,
        )

        try:
            clean_df, surviving_columns = filter_and_impute_columns(observation_df, max_gap=max_gap)
        except ValidationError as imputation_err:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(imputation_err))
            return inference_pb2.PredictHourlyWeatherResponse()

        target = clean_df[surviving_columns].to_numpy(dtype=float).T

        past_timestamps = observation_df["timestamp"].tolist()
        future_timestamps = generate_hourly_future_timestamps(issue_time_dt, request.horizon_hours)
        lat = request.latitude if request.HasField("latitude") else None

        past_covariates = calculate_temporal_covariates(past_timestamps, latitude=lat)
        future_covariates = calculate_temporal_covariates(future_timestamps, latitude=lat)

        max_recommended = self.backend.max_recommended_horizon()
        if max_recommended is not None and prediction_length > max_recommended:
            logger.warning(
                "prediction length %d exceeds the recommended maximum of %d for model %s; "
                "longer forecasts may degrade in quality",
                prediction_length,
                max_recommended,
                self.backend.model_name,
            )

        try:
            quantiles, _ = self.backend.predict(
                target=target,
                past_covariates=past_covariates,
                future_covariates=future_covariates,
                prediction_length=prediction_length,
            )
        except Exception as inference_err:
            logger.exception("inference execution failed")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"inference execution failed: {inference_err}")
            return inference_pb2.PredictHourlyWeatherResponse()

        predictions = format_hourly_predictions(
            node_id=request.node_id,
            issue_time=aligned_issue_time,
            horizon_hours=request.horizon_hours,
            quantiles=quantiles,
            target_columns=surviving_columns,
        )

        now = datetime.now(UTC)
        computed_at = Timestamp()
        computed_at.FromDatetime(now)

        return inference_pb2.PredictHourlyWeatherResponse(
            predictions=predictions,
            computed_at=computed_at,
        )

    def PredictDailyWeather(
        self,
        request: inference_pb2.PredictDailyWeatherRequest,
        context: grpc.ServicerContext,
    ) -> inference_pb2.PredictDailyWeatherResponse:
        """PredictDailyWeather runs ingress validation, feature transformations, and daily forecasting."""
        try:
            validate_daily_request(request)
        except ValidationError as validation_err:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(validation_err))
            return inference_pb2.PredictDailyWeatherResponse()

        prediction_length = request.horizon_days
        max_gap = MAX_GAP_DAILY

        issue_time_dt = request.issue_time.ToDatetime().replace(tzinfo=UTC)

        sorted_observations = sort_observations(request.history)
        observation_df = extract_observation_dataframe(
            sorted_observations,
            step=timedelta(days=1),
            issue_time=issue_time_dt,
        )

        try:
            clean_df, surviving_columns = filter_and_impute_columns(observation_df, max_gap=max_gap)
        except ValidationError as imputation_err:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(imputation_err))
            return inference_pb2.PredictDailyWeatherResponse()

        target = clean_df[surviving_columns].to_numpy(dtype=float).T

        past_timestamps = observation_df["timestamp"].tolist()
        future_timestamps = generate_daily_future_timestamps(issue_time_dt, request.horizon_days)
        lat = request.latitude if request.HasField("latitude") else None

        past_covariates = calculate_temporal_covariates(past_timestamps, latitude=lat)
        future_covariates = calculate_temporal_covariates(future_timestamps, latitude=lat)

        max_recommended = self.backend.max_recommended_horizon()
        if max_recommended is not None and prediction_length > max_recommended:
            logger.warning(
                "prediction length %d exceeds the recommended maximum of %d for model %s; "
                "longer forecasts may degrade in quality",
                prediction_length,
                max_recommended,
                self.backend.model_name,
            )

        try:
            quantiles, _ = self.backend.predict(
                target=target,
                past_covariates=past_covariates,
                future_covariates=future_covariates,
                prediction_length=prediction_length,
            )
        except Exception as inference_err:
            logger.exception("inference execution failed")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"inference execution failed: {inference_err}")
            return inference_pb2.PredictDailyWeatherResponse()

        predictions = format_daily_predictions(
            node_id=request.node_id,
            issue_time=request.issue_time,
            horizon_days=request.horizon_days,
            quantiles=quantiles,
            target_columns=surviving_columns,
        )

        now = datetime.now(UTC)
        computed_at = Timestamp()
        computed_at.FromDatetime(now)

        return inference_pb2.PredictDailyWeatherResponse(
            predictions=predictions,
            computed_at=computed_at,
        )

    def GetModelInfo(
        self,
        request: inference_pb2.GetModelInfoRequest,
        context: grpc.ServicerContext,
    ) -> inference_pb2.GetModelInfoResponse:
        """GetModelInfo returns active model identifier and maximum supported horizon."""
        return inference_pb2.GetModelInfoResponse(
            model_name=self.backend.model_name,
            max_horizon_hours=self.config.max_horizon_hours,
        )


class InferenceServer:
    """InferenceServer manages the gRPC server lifecycle and periodic idle model unloading."""

    def __init__(
        self,
        config: Config,
        backend: ForecastBackend | None = None,
    ):
        """InferenceServer initializes gRPC server and background tasks."""
        self.config = config
        self.backend = backend or create_backend(
            backend_name=config.model_backend,
            model_name=config.model_name,
            idle_timeout_seconds=config.idle_timeout_seconds,
        )
        self.servicer = InferenceServicer(self.backend, self.config)
        self._server = grpc.server(futures.ThreadPoolExecutor(max_workers=config.max_workers))
        inference_pb2_grpc.add_InferenceServiceServicer_to_server(self.servicer, self._server)
        self.health_servicer = health.HealthServicer()
        health_pb2_grpc.add_HealthServicer_to_server(self.health_servicer, self._server)
        self.bound_port = self._server.add_insecure_port(f"{config.host}:{config.port}")

        self._stop_event = threading.Event()
        self._idle_thread = threading.Thread(target=self._idle_monitor, daemon=True)

    def _idle_monitor(self) -> None:
        """_idle_monitor periodically triggers idle model eviction."""
        while not self._stop_event.wait(timeout=60.0):
            if self.backend.maybe_unload_idle():
                logger.info("idle model successfully evicted from memory")

    def start(self) -> None:
        """Start begins serving gRPC requests and starts the idle monitor thread."""
        self.health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
        self.health_servicer.set("ready", health_pb2.HealthCheckResponse.SERVING)
        self._server.start()
        self._idle_thread.start()
        logger.info("inference server started on %s:%d", self.config.host, self.config.port)

    def stop(self, grace: float = 5.0) -> None:
        """Stop shuts down the gRPC server and terminates the idle monitor thread."""
        self.health_servicer.enter_graceful_shutdown()
        self._stop_event.set()
        self._server.stop(grace)
        logger.info("inference server stopped")

    def wait_for_termination(self) -> None:
        """wait_for_termination blocks until the server terminates."""
        self._server.wait_for_termination()
