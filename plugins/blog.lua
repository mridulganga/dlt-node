function Run()
	s = recordStartTime()
	response = restGet("https://mridulganga.dev")
	latency = recordEndTime(s)
	-- log("latency"..tostring(latency))
	-- log("status "..tostring(response["statusCode"]))
	if response["statusCode"] ~= 200 then
		return buildResult("false", tostring(response["statusCode"]), tostring(latency), "error")
	end
	return buildResult("true", tostring(response["statusCode"]), tostring(latency), "success")
end