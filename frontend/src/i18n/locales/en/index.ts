import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import intelligence from './intelligence'
import feishu from './feishu'
import feishuKeyDelivery from './feishuKeyDelivery'
import poolDashboard from './poolDashboard'

export default {
  adminAccessKeys,
  mcp,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
  ...intelligence,
  feishu,
  feishuKeyDelivery,
  poolDashboard,
}
