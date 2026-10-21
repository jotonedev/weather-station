# Rain Gauge

Remix of https://smartsolutions4home.com/ss4h-rg-rain-gauge/

## Calibration

Use a syringe to measure how many milliliters of water are needed to trigger a bucket tip. Enter that measured value into `mcu/src/control-node/constants.h` as `RAIN_METER_BUCKET_VOLUME_ML`. The firmware uses the funnel radius and area formula in the configuration file to compute the rainfall depth per tip in millimeters.
