import { http } from './client'

export interface WeatherCurrent {
  time: string
  interval?: number
  temperature_2m: number
  weather_code: number
  is_day: 0 | 1
}

export interface WeatherResponse {
  latitude: number
  longitude: number
  timezone: string
  current_units?: Record<string, string>
  current: WeatherCurrent
}

export async function getWeather(lat: number, lon: number): Promise<WeatherResponse> {
  const { data } = await http.get<WeatherResponse>('/public/weather', {
    params: { lat, lon },
  })
  return data
}
