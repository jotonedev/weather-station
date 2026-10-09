// @generated
/// Generated client implementations.
pub mod weather_database_service_client {
    #![allow(
        unused_variables,
        dead_code,
        missing_docs,
        clippy::wildcard_imports,
        clippy::let_unit_value,
    )]
    use tonic::codegen::*;
    use tonic::codegen::http::Uri;
    #[derive(Debug, Clone)]
    pub struct WeatherDatabaseServiceClient<T> {
        inner: tonic::client::Grpc<T>,
    }
    impl WeatherDatabaseServiceClient<tonic::transport::Channel> {
        /// Attempt to create a new client by connecting to a given endpoint.
        pub async fn connect<D>(dst: D) -> Result<Self, tonic::transport::Error>
        where
            D: TryInto<tonic::transport::Endpoint>,
            D::Error: Into<StdError>,
        {
            let conn = tonic::transport::Endpoint::new(dst)?.connect().await?;
            Ok(Self::new(conn))
        }
    }
    impl<T> WeatherDatabaseServiceClient<T>
    where
        T: tonic::client::GrpcService<tonic::body::Body>,
        T::Error: Into<StdError>,
        T::ResponseBody: Body<Data = Bytes> + std::marker::Send + 'static,
        <T::ResponseBody as Body>::Error: Into<StdError> + std::marker::Send,
    {
        pub fn new(inner: T) -> Self {
            let inner = tonic::client::Grpc::new(inner);
            Self { inner }
        }
        pub fn with_origin(inner: T, origin: Uri) -> Self {
            let inner = tonic::client::Grpc::with_origin(inner, origin);
            Self { inner }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> WeatherDatabaseServiceClient<InterceptedService<T, F>>
        where
            F: tonic::service::Interceptor,
            T::ResponseBody: Default,
            T: tonic::codegen::Service<
                http::Request<tonic::body::Body>,
                Response = http::Response<
                    <T as tonic::client::GrpcService<tonic::body::Body>>::ResponseBody,
                >,
            >,
            <T as tonic::codegen::Service<
                http::Request<tonic::body::Body>,
            >>::Error: Into<StdError> + std::marker::Send + std::marker::Sync,
        {
            WeatherDatabaseServiceClient::new(
                InterceptedService::new(inner, interceptor),
            )
        }
        /// Compress requests with the given encoding.
        ///
        /// This requires the server to support it otherwise it might respond with an
        /// error.
        #[must_use]
        pub fn send_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.send_compressed(encoding);
            self
        }
        /// Enable decompressing responses.
        #[must_use]
        pub fn accept_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.accept_compressed(encoding);
            self
        }
        /// Limits the maximum size of a decoded message.
        ///
        /// Default: `4MB`
        #[must_use]
        pub fn max_decoding_message_size(mut self, limit: usize) -> Self {
            self.inner = self.inner.max_decoding_message_size(limit);
            self
        }
        /// Limits the maximum size of an encoded message.
        ///
        /// Default: `usize::MAX`
        #[must_use]
        pub fn max_encoding_message_size(mut self, limit: usize) -> Self {
            self.inner = self.inner.max_encoding_message_size(limit);
            self
        }
        pub async fn store_readings(
            &mut self,
            request: impl tonic::IntoRequest<super::StoreReadingsRequest>,
        ) -> std::result::Result<
            tonic::Response<super::StoreReadingsResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/StoreReadings",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "StoreReadings",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn store_predictions(
            &mut self,
            request: impl tonic::IntoRequest<super::StorePredictionsRequest>,
        ) -> std::result::Result<
            tonic::Response<super::StorePredictionsResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/StorePredictions",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "StorePredictions",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn create_node(
            &mut self,
            request: impl tonic::IntoRequest<super::CreateNodeRequest>,
        ) -> std::result::Result<
            tonic::Response<super::CreateNodeResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/CreateNode",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "CreateNode",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn update_node(
            &mut self,
            request: impl tonic::IntoRequest<super::UpdateNodeRequest>,
        ) -> std::result::Result<
            tonic::Response<super::UpdateNodeResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/UpdateNode",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "UpdateNode",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn get_readings(
            &mut self,
            request: impl tonic::IntoRequest<super::GetReadingsRequest>,
        ) -> std::result::Result<
            tonic::Response<tonic::codec::Streaming<super::GetReadingsResponse>>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/GetReadings",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "GetReadings",
                    ),
                );
            self.inner.server_streaming(req, path, codec).await
        }
        pub async fn get_latest_readings(
            &mut self,
            request: impl tonic::IntoRequest<super::GetLatestReadingsRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetLatestReadingsResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/GetLatestReadings",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "GetLatestReadings",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn get_forecast(
            &mut self,
            request: impl tonic::IntoRequest<super::GetForecastRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetForecastResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/GetForecast",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "GetForecast",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn get_node_coordinates(
            &mut self,
            request: impl tonic::IntoRequest<super::GetNodeCoordinatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetNodeCoordinatesResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/GetNodeCoordinates",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "GetNodeCoordinates",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn list_node_coordinates(
            &mut self,
            request: impl tonic::IntoRequest<super::ListNodeCoordinatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::ListNodeCoordinatesResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/ListNodeCoordinates",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "ListNodeCoordinates",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn get_climate_aggregates(
            &mut self,
            request: impl tonic::IntoRequest<super::GetClimateAggregatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetClimateAggregatesResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/GetClimateAggregates",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "GetClimateAggregates",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn get_air_quality_aggregates(
            &mut self,
            request: impl tonic::IntoRequest<super::GetAirQualityAggregatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetAirQualityAggregatesResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::unknown(
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic_prost::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/weather.database.v1.WeatherDatabaseService/GetAirQualityAggregates",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "weather.database.v1.WeatherDatabaseService",
                        "GetAirQualityAggregates",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
    }
}
/// Generated server implementations.
pub mod weather_database_service_server {
    #![allow(
        unused_variables,
        dead_code,
        missing_docs,
        clippy::wildcard_imports,
        clippy::let_unit_value,
    )]
    use tonic::codegen::*;
    /// Generated trait containing gRPC methods that should be implemented for use with WeatherDatabaseServiceServer.
    #[async_trait]
    pub trait WeatherDatabaseService: std::marker::Send + std::marker::Sync + 'static {
        async fn store_readings(
            &self,
            request: tonic::Request<super::StoreReadingsRequest>,
        ) -> std::result::Result<
            tonic::Response<super::StoreReadingsResponse>,
            tonic::Status,
        >;
        async fn store_predictions(
            &self,
            request: tonic::Request<super::StorePredictionsRequest>,
        ) -> std::result::Result<
            tonic::Response<super::StorePredictionsResponse>,
            tonic::Status,
        >;
        async fn create_node(
            &self,
            request: tonic::Request<super::CreateNodeRequest>,
        ) -> std::result::Result<
            tonic::Response<super::CreateNodeResponse>,
            tonic::Status,
        >;
        async fn update_node(
            &self,
            request: tonic::Request<super::UpdateNodeRequest>,
        ) -> std::result::Result<
            tonic::Response<super::UpdateNodeResponse>,
            tonic::Status,
        >;
        /// Server streaming response type for the GetReadings method.
        type GetReadingsStream: tonic::codegen::tokio_stream::Stream<
                Item = std::result::Result<super::GetReadingsResponse, tonic::Status>,
            >
            + std::marker::Send
            + 'static;
        async fn get_readings(
            &self,
            request: tonic::Request<super::GetReadingsRequest>,
        ) -> std::result::Result<
            tonic::Response<Self::GetReadingsStream>,
            tonic::Status,
        >;
        async fn get_latest_readings(
            &self,
            request: tonic::Request<super::GetLatestReadingsRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetLatestReadingsResponse>,
            tonic::Status,
        >;
        async fn get_forecast(
            &self,
            request: tonic::Request<super::GetForecastRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetForecastResponse>,
            tonic::Status,
        >;
        async fn get_node_coordinates(
            &self,
            request: tonic::Request<super::GetNodeCoordinatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetNodeCoordinatesResponse>,
            tonic::Status,
        >;
        async fn list_node_coordinates(
            &self,
            request: tonic::Request<super::ListNodeCoordinatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::ListNodeCoordinatesResponse>,
            tonic::Status,
        >;
        async fn get_climate_aggregates(
            &self,
            request: tonic::Request<super::GetClimateAggregatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetClimateAggregatesResponse>,
            tonic::Status,
        >;
        async fn get_air_quality_aggregates(
            &self,
            request: tonic::Request<super::GetAirQualityAggregatesRequest>,
        ) -> std::result::Result<
            tonic::Response<super::GetAirQualityAggregatesResponse>,
            tonic::Status,
        >;
    }
    #[derive(Debug)]
    pub struct WeatherDatabaseServiceServer<T> {
        inner: Arc<T>,
        accept_compression_encodings: EnabledCompressionEncodings,
        send_compression_encodings: EnabledCompressionEncodings,
        max_decoding_message_size: Option<usize>,
        max_encoding_message_size: Option<usize>,
    }
    impl<T> WeatherDatabaseServiceServer<T> {
        pub fn new(inner: T) -> Self {
            Self::from_arc(Arc::new(inner))
        }
        pub fn from_arc(inner: Arc<T>) -> Self {
            Self {
                inner,
                accept_compression_encodings: Default::default(),
                send_compression_encodings: Default::default(),
                max_decoding_message_size: None,
                max_encoding_message_size: None,
            }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> InterceptedService<Self, F>
        where
            F: tonic::service::Interceptor,
        {
            InterceptedService::new(Self::new(inner), interceptor)
        }
        /// Enable decompressing requests with the given encoding.
        #[must_use]
        pub fn accept_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.accept_compression_encodings.enable(encoding);
            self
        }
        /// Compress responses with the given encoding, if the client supports it.
        #[must_use]
        pub fn send_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.send_compression_encodings.enable(encoding);
            self
        }
        /// Limits the maximum size of a decoded message.
        ///
        /// Default: `4MB`
        #[must_use]
        pub fn max_decoding_message_size(mut self, limit: usize) -> Self {
            self.max_decoding_message_size = Some(limit);
            self
        }
        /// Limits the maximum size of an encoded message.
        ///
        /// Default: `usize::MAX`
        #[must_use]
        pub fn max_encoding_message_size(mut self, limit: usize) -> Self {
            self.max_encoding_message_size = Some(limit);
            self
        }
    }
    impl<T, B> tonic::codegen::Service<http::Request<B>>
    for WeatherDatabaseServiceServer<T>
    where
        T: WeatherDatabaseService,
        B: Body + std::marker::Send + 'static,
        B::Error: Into<StdError> + std::marker::Send + 'static,
    {
        type Response = http::Response<tonic::body::Body>;
        type Error = std::convert::Infallible;
        type Future = BoxFuture<Self::Response, Self::Error>;
        fn poll_ready(
            &mut self,
            _cx: &mut Context<'_>,
        ) -> Poll<std::result::Result<(), Self::Error>> {
            Poll::Ready(Ok(()))
        }
        fn call(&mut self, req: http::Request<B>) -> Self::Future {
            match req.uri().path() {
                "/weather.database.v1.WeatherDatabaseService/StoreReadings" => {
                    #[allow(non_camel_case_types)]
                    struct StoreReadingsSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::StoreReadingsRequest>
                    for StoreReadingsSvc<T> {
                        type Response = super::StoreReadingsResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::StoreReadingsRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::store_readings(
                                        &inner,
                                        request,
                                    )
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = StoreReadingsSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/StorePredictions" => {
                    #[allow(non_camel_case_types)]
                    struct StorePredictionsSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::StorePredictionsRequest>
                    for StorePredictionsSvc<T> {
                        type Response = super::StorePredictionsResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::StorePredictionsRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::store_predictions(
                                        &inner,
                                        request,
                                    )
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = StorePredictionsSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/CreateNode" => {
                    #[allow(non_camel_case_types)]
                    struct CreateNodeSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::CreateNodeRequest>
                    for CreateNodeSvc<T> {
                        type Response = super::CreateNodeResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::CreateNodeRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::create_node(&inner, request)
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = CreateNodeSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/UpdateNode" => {
                    #[allow(non_camel_case_types)]
                    struct UpdateNodeSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::UpdateNodeRequest>
                    for UpdateNodeSvc<T> {
                        type Response = super::UpdateNodeResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::UpdateNodeRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::update_node(&inner, request)
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = UpdateNodeSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/GetReadings" => {
                    #[allow(non_camel_case_types)]
                    struct GetReadingsSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::ServerStreamingService<super::GetReadingsRequest>
                    for GetReadingsSvc<T> {
                        type Response = super::GetReadingsResponse;
                        type ResponseStream = T::GetReadingsStream;
                        type Future = BoxFuture<
                            tonic::Response<Self::ResponseStream>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::GetReadingsRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::get_readings(&inner, request)
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = GetReadingsSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.server_streaming(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/GetLatestReadings" => {
                    #[allow(non_camel_case_types)]
                    struct GetLatestReadingsSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::GetLatestReadingsRequest>
                    for GetLatestReadingsSvc<T> {
                        type Response = super::GetLatestReadingsResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::GetLatestReadingsRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::get_latest_readings(
                                        &inner,
                                        request,
                                    )
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = GetLatestReadingsSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/GetForecast" => {
                    #[allow(non_camel_case_types)]
                    struct GetForecastSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::GetForecastRequest>
                    for GetForecastSvc<T> {
                        type Response = super::GetForecastResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::GetForecastRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::get_forecast(&inner, request)
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = GetForecastSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/GetNodeCoordinates" => {
                    #[allow(non_camel_case_types)]
                    struct GetNodeCoordinatesSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::GetNodeCoordinatesRequest>
                    for GetNodeCoordinatesSvc<T> {
                        type Response = super::GetNodeCoordinatesResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::GetNodeCoordinatesRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::get_node_coordinates(
                                        &inner,
                                        request,
                                    )
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = GetNodeCoordinatesSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/ListNodeCoordinates" => {
                    #[allow(non_camel_case_types)]
                    struct ListNodeCoordinatesSvc<T: WeatherDatabaseService>(pub Arc<T>);
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::ListNodeCoordinatesRequest>
                    for ListNodeCoordinatesSvc<T> {
                        type Response = super::ListNodeCoordinatesResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::ListNodeCoordinatesRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::list_node_coordinates(
                                        &inner,
                                        request,
                                    )
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = ListNodeCoordinatesSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/GetClimateAggregates" => {
                    #[allow(non_camel_case_types)]
                    struct GetClimateAggregatesSvc<T: WeatherDatabaseService>(
                        pub Arc<T>,
                    );
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::GetClimateAggregatesRequest>
                    for GetClimateAggregatesSvc<T> {
                        type Response = super::GetClimateAggregatesResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::GetClimateAggregatesRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::get_climate_aggregates(
                                        &inner,
                                        request,
                                    )
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = GetClimateAggregatesSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/weather.database.v1.WeatherDatabaseService/GetAirQualityAggregates" => {
                    #[allow(non_camel_case_types)]
                    struct GetAirQualityAggregatesSvc<T: WeatherDatabaseService>(
                        pub Arc<T>,
                    );
                    impl<
                        T: WeatherDatabaseService,
                    > tonic::server::UnaryService<super::GetAirQualityAggregatesRequest>
                    for GetAirQualityAggregatesSvc<T> {
                        type Response = super::GetAirQualityAggregatesResponse;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<
                                super::GetAirQualityAggregatesRequest,
                            >,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as WeatherDatabaseService>::get_air_quality_aggregates(
                                        &inner,
                                        request,
                                    )
                                    .await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let method = GetAirQualityAggregatesSvc(inner);
                        let codec = tonic_prost::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                _ => {
                    Box::pin(async move {
                        let mut response = http::Response::new(
                            tonic::body::Body::default(),
                        );
                        let headers = response.headers_mut();
                        headers
                            .insert(
                                tonic::Status::GRPC_STATUS,
                                (tonic::Code::Unimplemented as i32).into(),
                            );
                        headers
                            .insert(
                                http::header::CONTENT_TYPE,
                                tonic::metadata::GRPC_CONTENT_TYPE,
                            );
                        Ok(response)
                    })
                }
            }
        }
    }
    impl<T> Clone for WeatherDatabaseServiceServer<T> {
        fn clone(&self) -> Self {
            let inner = self.inner.clone();
            Self {
                inner,
                accept_compression_encodings: self.accept_compression_encodings,
                send_compression_encodings: self.send_compression_encodings,
                max_decoding_message_size: self.max_decoding_message_size,
                max_encoding_message_size: self.max_encoding_message_size,
            }
        }
    }
    /// Generated gRPC service name
    pub const SERVICE_NAME: &str = "weather.database.v1.WeatherDatabaseService";
    impl<T> tonic::server::NamedService for WeatherDatabaseServiceServer<T> {
        const NAME: &'static str = SERVICE_NAME;
    }
}
