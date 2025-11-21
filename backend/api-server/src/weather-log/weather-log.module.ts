import { Module } from '@nestjs/common';
import { WeatherLogService } from './weather-log.service';
import { WeatherLogController } from './weather-log.controller';

@Module({
  providers: [WeatherLogService],
  controllers: [WeatherLogController]
})
export class WeatherLogModule {}
