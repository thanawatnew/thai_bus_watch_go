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
