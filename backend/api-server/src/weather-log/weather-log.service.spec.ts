import { Test, TestingModule } from '@nestjs/testing';
import { WeatherLogService } from './weather-log.service';

describe('WeatherLogService', () => {
  let service: WeatherLogService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [WeatherLogService],
    }).compile();

    service = module.get<WeatherLogService>(WeatherLogService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
