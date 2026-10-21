# 3D Models

Printed enclosures and mechanical parts for the weather station sensors. Each subfolder provides the individual meshes in `stl/`, a pre-arranged `.3mf` project and the Fusion 360 file (`.f3d` or `.f3z`) to inspect how the parts fit together. 
Build photos are in [`../pictures/`](../pictures/).

## Assembly notes

Assembly mostly follows the upstream guides linked in each subfolder README. Every sensor enclosure connects back to the control node box over in-wall Cat5e cable without terminations. Sensor breakout boards are fixed in place with hot glue and soldered wire splices are insulated with heat shrink tubing.

## Calibration

- Rain gauge needs calibration to measure how much water triggers a tip
- Anemometer needs calibration to map rotation speed to actual wind speed
- Wind vane needs calibration to set the north direction offset

## Enclosures

### [`air-sensor-box/`](air-sensor-box/)

Stevenson screen enclosure that shields the air sensors (AHT20, BMP180, SGP30 and PMS7003) from sunlight and rain while keeping airflow open.

| Item | Quantity | Notes |
| --- | --- | --- |
| AHT20 breakout | 1 | Temperature and humidity |
| BMP180 breakout | 1 | Barometric pressure |
| SGP30 breakout | 1 | TVOC and eCO2 |
| PMS7003 | 1 | Particulate matter |
| M6 threaded rod | 1 | Support mount |
| M6 nut | 6 | Secures the M6 threaded rod |
| In-wall Cat5e cable (unterminated) | 1 run | Connects directly to the control node |
| Heat shrink tubing | As needed | Insulates soldered wire connections |

### [`anemometer-windvane/`](anemometer-windvane/)

Three-cup anemometer and wind vane mounted on a shared bracket. Both shafts hold a magnet read by an AS5600 sensor glued inside each housing.

| Item | Quantity | Notes |
| --- | --- | --- |
| AS5600 breakout | 2 | One for anemometer, one for wind vane |
| Diametrically magnetized magnet | 2 | Mounted in `supporto magnete` on each shaft |
| Ball bearings | 2 | Press-fit into bearing cages |
| M6 threaded rod | 1 | Support mount |
| M6 nut | 6 | Secures the M6 threaded rod |
| M4 threaded rod | 2 | Anemometer and wind vane shafts |
| M4 nut | 2 | Shaft assembly |
| M4 threadlock nut | 2 | Locks rotating shafts |
| In-wall Cat5e cable (unterminated) | 1 run | Connects directly to the control node |
| Heat shrink tubing | As needed | Insulates soldered wire connections |

### [`rain-gauge/`](rain-gauge/)

Tipping-bucket collector modified to use a magnet on the bucket axle and an AS5600 sensor inside the electronics box instead of a reed switch.

| Item | Quantity | Notes |
| --- | --- | --- |
| AS5600 breakout | 1 | Mounted inside the electronics box |
| Diametrically magnetized magnet | 1 | Mounted on the tipping-bucket axle |
| Ball bearings | 2 | Fit into the two `Roulement` housings |
| M4 threaded rod | 1 | Tipping-bucket axle |
| M4 bolt | 5 | Base and housing assembly |
| M4 nut | 5 | Secures the M4 bolts |
| In-wall Cat5e cable (unterminated) | 1 run | Connects directly to the control node |
| Heat shrink tubing | As needed | Insulates soldered wire connections |
