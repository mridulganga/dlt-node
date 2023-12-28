function Run()
	userResponse = rest_post("https://api/user", data)
	if userResponse.status != 200 then
		return response("failure", userResponse)
	end
	
	userGetResponse = rest_get("https://api/user?userId="..user["id"])
	if userGetResponse.status != 200 then
		return response("failure", userGetResponse)
	end
	
	userUpdateResponse = rest_update("https://api/user?userId="..user["id"], data)
	if userUpdateResponse.status != 200 then
		return response("failure", userUpdateResponse)
	end
	
	userDeleteResponse = rest_delete("https://api/user?userId="..user["id"])
	if userDeleteResponse == true {
		return response("failure", userDeleteResponse)
	}
	
	return result(true, userResponse.status, latency, userResponse.body)
end