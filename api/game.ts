import { AssetIndex, VersionMeta, VersionManifest } from "../types.ts";

/** Downloads the Minecraft version manifest, containing the URL and ID for each version. */
export async function getVersionManifest() {
  return <VersionManifest>(
    await (
      await fetch(
        "https://launchermeta.mojang.com/mc/game/version_manifest.json",
      )
    ).json()
  );
}

/**
Get the data for a specific version. This includes all the information needed to launch the game.
*/
export async function getVersionMeta(version: string) {
  const data = await getVersionManifest();
  const release = data.versions.find((element) => element.id == version);
  if (!release) {
    throw Error("Invalid version");
  }
  return <VersionMeta>await (await fetch(release.url)).json();
}

/** Using the version metadata, get the game asset data. */
export async function getAssetData(versionData: VersionMeta) {
  const url = versionData.assetIndex.url;
  const data = await (await fetch(url)).json();
  return <AssetIndex>data;
}
