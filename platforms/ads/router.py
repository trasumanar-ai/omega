"""Ads platform — simüle reklam servisi.

Discit'in SimAdsProvider'ı şu endpointleri çağırır:
  GET    /ads/campaigns
  GET    /ads/campaigns/{id}
  PATCH  /ads/campaigns/{id}
  GET    /ads/ad-sets
  GET    /ads/ad-sets/{id}
  GET    /ads/ads
  GET    /ads/ads/{id}
  GET    /ads/forms/{form_id}/leads
  GET    /ads/insights/{object_id}
  POST   /ads/conversions/{pixel_id}
"""
from fastapi import APIRouter

router = APIRouter()

_events: list[dict] = []


@router.get("/campaigns")
async def list_campaigns(account_id: str = "", status: str | None = None):
    return []


@router.get("/campaigns/{campaign_id}")
async def get_campaign(campaign_id: str):
    return {"id": campaign_id, "name": "sim-campaign", "status": "ACTIVE"}


@router.patch("/campaigns/{campaign_id}")
async def update_campaign(campaign_id: str):
    return {"id": campaign_id, "name": "sim-campaign", "status": "ACTIVE"}


@router.get("/ad-sets")
async def list_ad_sets(campaign_id: str = ""):
    return []


@router.get("/ad-sets/{ad_set_id}")
async def get_ad_set(ad_set_id: str):
    return {"id": ad_set_id, "campaign_id": "sim", "name": "sim-adset", "status": "ACTIVE"}


@router.get("/ads")
async def list_ads(ad_set_id: str = ""):
    return []


@router.get("/ads/{ad_id}")
async def get_ad(ad_id: str):
    return {"id": ad_id, "ad_set_id": "sim", "name": "sim-ad", "status": "ACTIVE"}


@router.get("/forms/{form_id}/leads")
async def get_leads(form_id: str):
    return []


@router.get("/insights/{object_id}")
async def get_insights(object_id: str):
    return []


@router.post("/conversions/{pixel_id}")
async def send_conversion(pixel_id: str, body: dict = {}):
    _events.append({"pixel_id": pixel_id, "data": body})
    return {"ok": True}


@router.get("/events")
async def list_events():
    """Debug — conversion eventleri."""
    return _events
