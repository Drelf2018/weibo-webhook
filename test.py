import httpx
httpx.post("http://localhost:8080/update", data={"mid": 2, "time": 3, "text": "测试"})