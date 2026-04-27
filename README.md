# thai_bus_watch_go
hope I can get this to integrate to Namtang after 10 years and AI shenanigan.

# How to use
You have to go to 
https://namtang.otp.go.th/

select bus stop and select bus number route, have to open inspect Network from Namtang for now and get trip id like

https://namtang-api.otp.go.th/front/trip/669?locale=en

and run:

go run main.go -trip-id 669 -bus 12-2543

I get help from OpenGPT 5.5 this time, hope I can get where I want and get people out of poverty.

example output:
```
thanawatnew@bazzitenkmhcool:/var/home/thanawatnew/Downloads/thai_bus_watch_go$ go run main.go -trip-id 669 -bus 12-2492

Telegram env not set, printing result only:
--------------------------------------------------
🚌 Bus found

Trip ID: 669
Bus number: 12-2492
Full ID: 12-2492 กรุงเทพมหานคร

📍 Bus GPS
Lat: 13.770562
Lon: 100.483955

🎥 Nearest BMA camera
Camera ID: 1379
Name: ทางลง ด่วนยกระดับฝั่งปิ่นเกล้า
Direction: ทางลง ด่วนยกระดับฝั่งปิ่นเกล้า
Camera lat: 13.769930
Camera lon: 100.484640
Distance: 102 meters

Camera feed:
http://www.bmatraffic.com/PlayVideo.aspx?ID=1379
--------------------------------------------------
{
  "bus": {
    "trip_id": "669",
    "bus_number": "12-2492",
    "full_id": "12-2492 กรุงเทพมหานคร",
    "lat": 13.770561666667,
    "lon": 100.483955,
    "raw": {
      "created": 1777263493,
      "distance_from_prev_stop": "741.7962753117116",
      "distance_to_next_stop": "64.55007548958093",
      "heading": "-42",
      "id": "12-2492 กรุงเทพมหานคร",
      "is_approaching_stop": false,
      "is_first_to_arrive": false,
      "is_outside_stop_range": false,
      "is_reversed": false,
      "lat": "13.770561666667",
      "lon": "100.483955",
      "next_stop_id": 2303,
      "next_stop_name": "ตรงข้ามพาต้าปิ่นเกล้า",
      "prev_stop_id": 2302,
      "prev_stop_name": "หลังสะพานพระปิ่นเกล้า",
      "received": 1777263493,
      "snapped_heading": 320,
      "snapped_lat": "13.770607574313392",
      "snapped_lon": "100.48401012318132",
      "speed": 7,
      "time": 1777263486
    }
  },
  "nearest_camera": {
    "id": "1379",
    "name_th": "ทางลง ด่วนยกระดับฝั่งปิ่นเกล้า",
    "name_en": "-",
    "direction_th": "ทางลง ด่วนยกระดับฝั่งปิ่นเกล้า",
    "direction_en": "-",
    "lat": 13.76993,
    "lon": 100.48464,
    "ip": "10.151.101.29",
    "icon": "pin-right.png",
    "feed_url": "http://www.bmatraffic.com/PlayVideo.aspx?ID=1379"
  },
  "distance_meters": 102,
  "telegram_sent": false
}
```
