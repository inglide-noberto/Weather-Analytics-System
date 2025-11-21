import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { WeatherLogModule } from './weather-log/weather-log.module';
import { AuthModule } from './auth/auth.module';
import { UsersModule } from './users/users.module';

@Module({
  imports: [WeatherLogModule, AuthModule, UsersModule],
  controllers: [AppController, ],
  providers: [AppService, ],
})
export class AppModule {}
