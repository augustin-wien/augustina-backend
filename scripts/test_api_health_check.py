import unittest
from unittest.mock import Mock

import api_health_check as api


def _resp(status: int, text: str = ""):
    r = Mock()
    r.status_code = status
    r.text = text
    return r


class ApiHealthCheckTests(unittest.TestCase):
    def test_check_health_ok(self):
        session = Mock()
        session.get.return_value = _resp(200, "ok")

        ok, msg = api.check_health(session, "http://localhost:3000")

        self.assertTrue(ok)
        self.assertEqual("OK", msg)
        # Check that the endpoint is called correctly
        call_args = session.get.call_args
        self.assertEqual("http://localhost:3000/healthz", call_args[0][0])

    def test_check_ready_failure(self):
        session = Mock()
        session.get.return_value = _resp(503, "not ready")

        ok, msg = api.check_ready(session, "http://localhost:3000")

        self.assertFalse(ok)
        self.assertIn("503", msg)
        # Check that the endpoint is called correctly
        call_args = session.get.call_args
        self.assertEqual("http://localhost:3000/readyz", call_args[0][0])

    def test_check_flour_vendor_unauthorized(self):
        session = Mock()
        session.get.return_value = _resp(401, "auth required")

        ok, msg = api.check_flour_vendor(
            session, "https://api.example.com/", "license-123", token="TOKEN"
        )

        self.assertFalse(ok)
        self.assertEqual("Unauthorized", msg)
        # Check that the endpoint is called correctly with token
        call_args = session.get.call_args
        self.assertEqual(
            "https://api.example.com/api/flour/vendors/license/license-123/",
            call_args[0][0],
        )
        self.assertIn("Authorization", call_args[1]["headers"])


if __name__ == "__main__":
    unittest.main()
