import asyncio
import re
from pathlib import Path
from urllib.parse import unquote, urljoin, urlparse

import bs4
import httpx

BASE_URL = "https://www.ubisoft.com"
MAPS_URL = f"{BASE_URL}/en-us/game/rainbow-six/siege/game-info/maps"
DOWNLOAD_DIR = Path("blueprints")


def parse_map_links(html: str) -> list[str]:
    soup = bs4.BeautifulSoup(html, "html.parser")
    map_links: list[str] = []

    for map_card in soup.find_all("a", class_="maplist__card"):
        href = map_card.get("href")
        if isinstance(href, str):
            map_links.append(urljoin(BASE_URL, href))

    return map_links


def parse_blueprint_link(html: str, page_url: str) -> str | None:
    soup = bs4.BeautifulSoup(html, "html.parser")
    blueprint_link = soup.find("a", class_="map-details__gallery__button")
    if not isinstance(blueprint_link, bs4.Tag):
        return None

    href = blueprint_link.get("href")
    if not isinstance(href, str):
        return None

    return urljoin(page_url, href)


def filename_from_url(url: str) -> str:
    parsed_url = urlparse(url)
    filename = Path(unquote(parsed_url.path)).name
    if not filename:
        filename = "blueprint.zip"
    if not filename.lower().endswith(".zip"):
        filename = f"{filename}.zip"

    return re.sub(r'[<>:"/\\|?*]', "_", filename)


async def get_map_links(client: httpx.AsyncClient) -> list[str]:
    response = await client.get(MAPS_URL)
    _ = response.raise_for_status()
    return parse_map_links(response.text)


async def get_blueprint_link(client: httpx.AsyncClient, map_link: str) -> str | None:
    response = await client.get(map_link)
    _ = response.raise_for_status()
    return parse_blueprint_link(response.text, map_link)


async def get_blueprint_links(client: httpx.AsyncClient, map_links: list[str]) -> list[str]:
    results = await asyncio.gather(
        *(get_blueprint_link(client, map_link) for map_link in map_links)
    )
    return [link for link in results if link is not None]


async def download_zip(
    client: httpx.AsyncClient, blueprint_link: str, download_dir: Path
) -> Path:
    download_dir.mkdir(parents=True, exist_ok=True)
    destination = download_dir / filename_from_url(blueprint_link)

    async with client.stream("GET", blueprint_link) as response:
        _ = response.raise_for_status()
        with destination.open("wb") as file:
            async for chunk in response.aiter_bytes():
                _ = file.write(chunk)

    return destination


async def download_blueprints(
    client: httpx.AsyncClient, blueprint_links: list[str], download_dir: Path
) -> list[Path]:
    return await asyncio.gather(
        *(download_zip(client, link, download_dir) for link in blueprint_links)
    )


async def main() -> None:
    async with httpx.AsyncClient(
        follow_redirects=True,
        timeout=30.0,
        headers={"User-Agent": "r6-maps-scrape/0.1.0"},
    ) as client:
        map_links = await get_map_links(client)
        print(f"Found {len(map_links)} maps")

        blueprint_links = await get_blueprint_links(client, map_links)
        print(f"Found {len(blueprint_links)} blueprint zips")

        downloaded_files = await download_blueprints(
            client, blueprint_links, DOWNLOAD_DIR
        )
        for downloaded_file in downloaded_files:
            print(f"Downloaded {downloaded_file}")


if __name__ == "__main__":
    asyncio.run(main())
