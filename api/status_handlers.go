/*
This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package api

import (
	"github.com/gofiber/fiber/v2"
	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"
)

func Status(c *fiber.Ctx, cfg *cfg.Config) error {
	res := &common.Response{}
	res.Success = true
	res.Code = fiber.StatusOK
	res.Message = "up and running"
	return c.Status(fiber.StatusOK).JSON(res)
}
