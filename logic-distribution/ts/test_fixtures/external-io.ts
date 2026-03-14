/** Test: External IO → EXTERNAL_IO */
import fs from "fs/promises";
import axios from "axios";

export async function fetchWeather(city: string) {
  const response = await axios.get(`https://api.weather.com/${city}`);
  return response.data;
}

export async function readConfig(configPath: string) {
  const content = await fs.readFile(configPath, "utf-8");
  return JSON.parse(content);
}

export async function writeReport(data: any, outputPath: string) {
  await fs.writeFile(outputPath, JSON.stringify(data, null, 2));
}

export async function callExternalApi() {
  const response = await fetch("https://api.example.com/data");
  return response.json();
}
