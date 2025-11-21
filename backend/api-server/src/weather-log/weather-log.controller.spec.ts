import { Test, TestingModule } from '@nestjs/testing';
import { WeatherLogController } from './weather-log.controller';

describe('WeatherLogController', () => {
  let controller: WeatherLogController;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [WeatherLogController],
    }).compile();

    controller = module.get<WeatherLogController>(WeatherLogController);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });
});
