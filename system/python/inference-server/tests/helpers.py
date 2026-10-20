"""Shared factories for lightweight in-memory model objects used in tests."""


def create_synthetic_chronos_pipeline():
    """create_synthetic_chronos_pipeline builds a small in-memory Chronos2Pipeline for CPU testing."""
    from chronos import Chronos2Pipeline
    from chronos.chronos2.config import Chronos2CoreConfig
    from chronos.chronos2.model import Chronos2Model

    core_config = Chronos2CoreConfig(d_model=32, d_kv=8, d_ff=64, num_layers=1, num_heads=2)
    all_quantiles = [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9]
    core_config.chronos_config = {
        "context_length": 64,
        "output_patch_size": 8,
        "input_patch_size": 8,
        "input_patch_stride": 8,
        "quantiles": all_quantiles,
        "use_reg_token": True,
    }
    model = Chronos2Model(core_config)
    return Chronos2Pipeline(model)


def create_synthetic_timestral_forecaster():
    """create_synthetic_timestral_forecaster builds a small in-memory TimestralForecaster for CPU testing."""
    import torch
    from timestral import TimestralConfig, TimestralForecaster, TimestralModel

    knots = (0.05, 0.1, 0.25, 0.4, 0.5, 0.6, 0.75, 0.9, 0.95)
    config = TimestralConfig(
        patch_size=8,
        d_model=32,
        num_heads=4,
        mlp_hidden_dim=64,
        block_schedule=("time", "time_variate", "time", "time_variate"),
        quantile_knots=knots,
        block_size=4,
        context_length=256,
    )
    torch.manual_seed(0)
    model = TimestralModel(config)
    for param in model.parameters():
        torch.nn.init.normal_(param, std=0.05)
    return TimestralForecaster(model.eval())
